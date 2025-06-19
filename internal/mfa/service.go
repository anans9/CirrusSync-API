package mfa

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"cirrussync-api/internal/logger"
	"cirrussync-api/pkg/config"
	"cirrussync-api/pkg/redis"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/argon2"
)

const (
	// Redis key prefixes
	emailTokenPrefix         = "mfa:email:token:"        // Stores email verification tokens
	emailVerifiedPrefix      = "mfa:email:verified:"     // Tracks verified emails
	totpSecretPrefix         = "mfa:totp:secret:"        // Stores TOTP secrets
	totpEnabledPrefix        = "mfa:totp:enabled:"       // Tracks enabled TOTP
	totpRecoveryPrefix       = "mfa:totp:recovery:"      // Stores recovery keys
	emailRateLimitPrefix     = "mfa:rate:email:"         // Email rate limiting by email
	intentRateLimitPrefix    = "mfa:rate:intent:"        // Email rate limiting by intent
	intentByEmailPrefix      = "mfa:intent:email:"       // Track intents by email
	lastEmailSentTimePrefix  = "mfa:last_sent:email:"    // Last time email was sent
	emailCountPrefix         = "mfa:count:email:"        // Count of emails sent
	emailCountByIntentPrefix = "mfa:count:intent:email:" // Count of emails sent by intent

	// Hash settings
	argon2Memory      = 64 * 1024
	argon2Iterations  = 3
	argon2Parallelism = 2
	argon2KeyLength   = 32
	argon2SaltLength  = 16

	// Default expiration times
	defaultTokenExpiry    = 10 * time.Minute     // Email verification token expiration
	emailVerifiedExpiry   = 30 * time.Hour       // How long to keep email verification status
	totpSecretExpiry      = 365 * 24 * time.Hour // Permanent storage for TOTP secrets
	singleEmailExpiry     = 3 * time.Minute      // Minimum time between emails to same address
	windowRateLimitExpiry = 2 * time.Hour        // Window for rate limiting (5 emails per 2 hours)

	// Rate limits
	maxEmailsPerWindow = 5 // Maximum emails per time window
)

// EmailVerificationResult contains the result of a verification email request
type EmailVerificationResult struct {
	RemainingRequests int    // Remaining requests in the current window
	ResetTime         string // Time when the rate limit window resets (human readable)
	NextAllowedTime   string // Next time when an email can be sent (human readable)
}

// Config holds the MFA service configuration
type MFAConfig struct {
	config.MailConfig
	config.TOTPConfig
}

// TOTPData contains TOTP setup information
type TOTPData struct {
	Secret       string   `json:"secret"`
	QRCodeURL    string   `json:"qrCodeUrl"`
	RecoveryKeys []string `json:"recoveryKeys,omitempty"`
}

// SMTPClient represents a reusable SMTP client connection
type SMTPClient struct {
	client *smtp.Client
	auth   smtp.Auth
	from   string
	mu     sync.Mutex
}

// SMTPClientPool manages a pool of SMTP client connections
type SMTPClientPool struct {
	pool      []*SMTPClient
	config    *MFAConfig
	available chan *SMTPClient
	mu        sync.Mutex
}

var (
	// Global pool instance
	smtpPool     *SMTPClientPool
	smtpPoolOnce sync.Once
)

// Service handles MFA operations
type Service struct {
	config      MFAConfig
	repo        Repository
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewService creates a new MFA service
func NewService(repo Repository,
	config MFAConfig, redisClient *redis.Client, logger *logger.Logger) *Service {
	// Set default token expiry if not provided
	if config.TokenExpiry == 0 {
		config.TokenExpiry = defaultTokenExpiry
	}

	// Set TOTP defaults
	if config.TOTPIssuer == "" {
		config.TOTPIssuer = "CirrusSync"
	}
	if config.TOTPDigits == 0 {
		config.TOTPDigits = otp.DigitsSix
	}
	if config.TOTPPeriod == 0 {
		config.TOTPPeriod = 30
	}
	if config.TOTPSkew == 0 {
		config.TOTPSkew = 1
	}
	if config.TOTPAlgorithm == 0 {
		config.TOTPAlgorithm = otp.AlgorithmSHA1
	}

	service := &Service{
		config:      config,
		repo:        repo,
		redisClient: redisClient,
		logger:      logger,
	}

	// Initialize SMTP pool
	smtpPoolOnce.Do(func() {
		smtpPool = initSMTPPool(&config, 5) // Pool size of 5
	})

	return service
}

// initSMTPPool initializes the SMTP client pool
func initSMTPPool(config *MFAConfig, poolSize int) *SMTPClientPool {
	pool := &SMTPClientPool{
		pool:      make([]*SMTPClient, 0, poolSize),
		config:    config,
		available: make(chan *SMTPClient, poolSize),
		mu:        sync.Mutex{},
	}

	// Pre-create connections
	for range make([]struct{}, poolSize) {
		client, err := pool.createClient()
		if err != nil {
			// Log error but continue - we'll create more connections as needed
			continue
		}
		pool.available <- client
	}

	// Start a goroutine to periodically refresh connections
	go pool.maintainConnections()

	return pool
}

// createClient creates a new SMTP client connection
func (p *SMTPClientPool) createClient() (*SMTPClient, error) {
	// Connect to the server
	smtpAddr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)

	// Create the SMTP client
	client, err := smtp.Dial(smtpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SMTP server: %w", err)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: p.config.SMTPHost,
	}

	// Start TLS
	if err = client.StartTLS(tlsConfig); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	auth := smtp.PlainAuth("", p.config.SMTPUsername, p.config.SMTPPassword, p.config.SMTPHost)
	if err = client.Auth(auth); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return &SMTPClient{
		client: client,
		auth:   auth,
		from:   p.config.FromEmail,
		mu:     sync.Mutex{},
	}, nil
}

// getClient gets an available SMTP client from the pool or creates a new one
func (p *SMTPClientPool) getClient() (*SMTPClient, error) {
	// Try to get an existing client
	select {
	case client := <-p.available:
		return client, nil
	default:
		// No available clients, create a new one
		return p.createClient()
	}
}

// releaseClient returns a client to the pool
func (p *SMTPClientPool) releaseClient(client *SMTPClient) {
	// Only return working clients to the pool
	if client == nil || client.client == nil {
		return
	}

	// Try to return to pool or close if pool is full
	select {
	case p.available <- client:
		// Successfully returned to pool
	default:
		// Pool is full, close this connection
		client.client.Close()
	}
}

// maintainConnections periodically refreshes connections
func (p *SMTPClientPool) maintainConnections() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		poolSize := cap(p.available)
		p.mu.Unlock()

		// Check current pool size
		currentSize := len(p.available)
		if currentSize < poolSize/2 {
			// Replenish pool if it's less than half full
			for i := 0; i < poolSize-currentSize; i++ {
				client, err := p.createClient()
				if err != nil {
					// Log error but continue
					continue
				}
				select {
				case p.available <- client:
					// Added to pool
				default:
					// Pool is somehow full now, close connection
					client.client.Close()
					break
				}
			}
		}
	}
}

// generateToken creates a secure random token for verification
func generateToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", &TokenGenerationError{Err: err}
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SendVerificationEmail sends an email with a verification link
func (s *Service) SendVerificationEmail(ctx context.Context, email string, username string, intent string) (*EmailVerificationResult, error) {
	// Validate input
	if email == "" {
		return nil, ErrInvalidEmail
	}

	if !isValidIntent(intent) {
		return nil, ErrInvalidInput
	}

	// Normalize email
	email = NormalizeEmail(email)

	// Validate email format
	if !ValidateEmail(email) {
		return nil, ErrInvalidEmail
	}

	// Create result info to return
	result := &EmailVerificationResult{}

	// Check "user exists" logic based on intent
	if intent == "signup" {
		// Check if email already exists
		if userByEmail, err := s.repo.FindUserOneWhere(&email, nil); err != nil {
			s.logger.Error("Failed to check email existence", "email", email, "error", err)
		} else if userByEmail.Email != "" {
			return result, ErrEmailExists
		}

		// Check if username already exists
		if userByUsername, err := s.repo.FindUserOneWhere(nil, &username); err != nil {
			s.logger.Error("Failed to check username existence", "username", username, "error", err)
		} else if userByUsername.Email != "" {
			return result, ErrUsernameExists
		}
	}

	// Check if email is already verified for signup intent
	if intent == "signup" {
		verifiedKey := emailVerifiedPrefix + email
		isVerified, err := s.redisClient.SIsMember(ctx, verifiedKey, "true")
		if err != nil {
			s.logger.Error("Failed to check if email is verified", "email", email, "error", err)
		} else if isVerified {
			// Email is already verified, for signup this is an error
			return result, ErrEmailExists
		}
	}

	// Apply rate limiting logic
	canSend, nextAllowed, err := s.canSendEmail(ctx, email, intent)
	if err != nil {
		s.logger.Error("Failed to check rate limit", "email", email, "intent", intent, "error", err)
		// Fall through and try to proceed anyway
	} else if !canSend {
		// Format next allowed time for response
		if !nextAllowed.IsZero() {
			result.NextAllowedTime = s.formatTimeRemaining(time.Until(nextAllowed))
		}
		return result, ErrRateLimitExceeded
	}

	// Use distributed lock to prevent duplicate emails
	lockName := fmt.Sprintf("email_verification:%s:%s", email, intent)
	acquired, err := s.redisClient.AcquireLock(ctx, lockName, 30*time.Second, 3, 100*time.Millisecond)
	if err != nil {
		s.logger.Error("Error acquiring lock for email verification", "email", email, "intent", intent, "error", err)
	} else if !acquired {
		// Calculate next allowed time for recently sent emails
		lastSentKey := lastEmailSentTimePrefix + email + ":" + intent
		lastSentStr, err := s.redisClient.Get(ctx, lastSentKey)
		if err == nil && lastSentStr != "" {
			var lastSent time.Time
			if err := lastSent.UnmarshalText([]byte(lastSentStr)); err == nil {
				nextAllowed = lastSent.Add(singleEmailExpiry)
				if time.Now().Before(nextAllowed) {
					result.NextAllowedTime = s.formatTimeRemaining(time.Until(nextAllowed))
				}
			}
		}

		s.logger.Warn("Email verification already in progress", "email", email, "intent", intent)
		return result, ErrEmailAlreadySent
	}

	// Get remaining requests and reset time for the response
	windowEnd, remainingRequests, err := s.getRateLimitInfo(ctx, email)
	if err == nil {
		result.RemainingRequests = remainingRequests
		result.ResetTime = s.formatTimeRemaining(time.Until(windowEnd))
	}

	// Launch async email sending
	go func() {
		// Create a background context for the goroutine
		asyncCtx := context.Background()

		defer func() {
			// Release lock when done
			_, err := s.redisClient.ReleaseLock(asyncCtx, lockName)
			if err != nil {
				s.logger.Error("Failed to release lock", "lock", lockName, "error", err)
			}
		}()

		// Generate token
		token, err := generateToken()
		if err != nil {
			s.logger.Error("Failed to generate token", "error", err)
			return
		}

		// Store token in Redis with intent and email
		tokenData := fmt.Sprintf("%s:%s:%s", email, username, intent)
		tokenKey := emailTokenPrefix + token
		err = s.redisClient.Set(asyncCtx, tokenKey, tokenData, s.config.TokenExpiry)
		if err != nil {
			s.logger.Error("Failed to store token in Redis", "error", err)
			return
		}

		// Track this email sending
		err = s.trackEmailSent(asyncCtx, email, intent)
		if err != nil {
			s.logger.Error("Failed to track email sending", "error", err)
			// Continue anyway
		}

		// Create verification URL
		verificationURL := fmt.Sprintf("%s/verify?token=%s", s.config.BaseURL, token)

		// Get email content
		subject, htmlBody, textBody := s.getEmailContent(intent, verificationURL, username, s.formatDuration(s.config.TokenExpiry))

		// Send email
		err = s.sendEmailFast([]string{email}, subject, htmlBody, textBody)
		if err != nil {
			s.logger.Error("Failed to send verification email", "email", email, "intent", intent, "error", err)
		} else {
			s.logger.Info("Verification email sent", "email", email, "intent", intent)
		}
	}()

	return result, nil
}

// GetInt retrieves an integer value from Redis
func (s *Service) GetInt(ctx context.Context, key string) (int, error) {
	value, err := s.redisClient.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	if value == "" {
		return 0, nil
	}

	var result int
	_, err = fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

// Increment increments a counter in Redis
func (s *Service) Increment(ctx context.Context, key string) (int64, error) {
	// For each key, try to get current value
	val, err := s.redisClient.Get(ctx, key)
	if err != nil {
		// If error, start from 1
		err = s.redisClient.Set(ctx, key, "1", 0)
		if err != nil {
			return 0, err
		}
		return 1, nil
	}

	// Parse current value
	var current int64
	if val != "" {
		_, err = fmt.Sscanf(val, "%d", &current)
		if err != nil {
			return 0, err
		}
	}

	// Increment
	current++
	err = s.redisClient.Set(ctx, key, fmt.Sprintf("%d", current), 0)
	if err != nil {
		return 0, err
	}

	return current, nil
}

// canSendEmail checks if an email can be sent based on rate limits
func (s *Service) canSendEmail(ctx context.Context, email, intent string) (bool, time.Time, error) {
	// Check single email cooldown period
	lastSentKey := lastEmailSentTimePrefix + email + ":" + intent
	lastSentStr, err := s.redisClient.Get(ctx, lastSentKey)

	if err == nil && lastSentStr != "" {
		var lastSent time.Time
		if err := lastSent.UnmarshalText([]byte(lastSentStr)); err == nil {
			// Check if the cooldown period has passed
			nextAllowed := lastSent.Add(singleEmailExpiry)
			if time.Now().Before(nextAllowed) {
				return false, nextAllowed, nil
			}
		}
	}

	// Check window rate limit (e.g., 5 emails per 2 hours)
	emailCountKey := emailCountByIntentPrefix + email + ":" + intent
	count, err := s.GetInt(ctx, emailCountKey)
	if err == nil && count >= maxEmailsPerWindow {
		// Get when the window expires
		ttl, err := s.redisClient.TTL(ctx, emailCountKey)
		if err == nil && ttl > 0 {
			resetTime := time.Now().Add(ttl)
			return false, resetTime, nil
		}
		return false, time.Time{}, nil
	}

	// Check total email count (across all intents)
	totalEmailCountKey := emailCountPrefix + email
	totalCount, err := s.GetInt(ctx, totalEmailCountKey)
	if err == nil && totalCount >= maxEmailsPerWindow*2 { // Double the limit for all intents combined
		// Get when the window expires
		ttl, err := s.redisClient.TTL(ctx, totalEmailCountKey)
		if err == nil && ttl > 0 {
			resetTime := time.Now().Add(ttl)
			return false, resetTime, nil
		}
		return false, time.Time{}, nil
	}

	return true, time.Time{}, nil
}

// getRateLimitInfo returns the rate limit information for an email
func (s *Service) getRateLimitInfo(ctx context.Context, email string) (time.Time, int, error) {
	// Get the count of emails sent in this window
	totalEmailCountKey := emailCountPrefix + email
	count, err := s.GetInt(ctx, totalEmailCountKey)
	if err != nil {
		return time.Time{}, 0, err
	}

	// Get when the window expires
	ttl, err := s.redisClient.TTL(ctx, totalEmailCountKey)
	if err != nil {
		return time.Time{}, 0, err
	}

	// Calculate the window end time
	windowEnd := time.Now().Add(ttl)
	remainingRequests := maxEmailsPerWindow*2 - count

	if remainingRequests < 0 {
		remainingRequests = 0
	}

	return windowEnd, remainingRequests, nil
}

// trackEmailSent updates Redis to track that an email was sent
func (s *Service) trackEmailSent(ctx context.Context, email, intent string) error {
	// Record time of last email sent
	now := time.Now()
	nowStr, err := now.MarshalText()
	if err != nil {
		return err
	}

	// Update last sent time
	lastSentKey := lastEmailSentTimePrefix + email + ":" + intent
	err = s.redisClient.Set(ctx, lastSentKey, string(nowStr), singleEmailExpiry)
	if err != nil {
		return err
	}

	// Increment per-intent counter
	emailCountKey := emailCountByIntentPrefix + email + ":" + intent
	_, err = s.Increment(ctx, emailCountKey)
	if err != nil {
		return err
	}
	// Set expiry if not already set
	s.redisClient.Expire(ctx, emailCountKey, windowRateLimitExpiry)

	// Increment total email counter
	totalEmailCountKey := emailCountPrefix + email
	_, err = s.Increment(ctx, totalEmailCountKey)
	if err != nil {
		return err
	}
	// Set expiry if not already set
	s.redisClient.Expire(ctx, totalEmailCountKey, windowRateLimitExpiry)

	// Store the intent used
	intentKey := intentByEmailPrefix + email
	s.redisClient.SAdd(ctx, intentKey, intent)
	s.redisClient.Expire(ctx, intentKey, windowRateLimitExpiry)

	return nil
}

// getEmailContent returns email content (subject, HTML and text) based on intent
func (s *Service) getEmailContent(intent, verificationURL, username, expiry string) (string, string, string) {
	var subject string
	var htmlTemplate string
	var textTemplate string

	switch intent {
	case "signup":
		subject = "Please verify your email address - CirrusSync"

		// HTML version
		htmlTemplate = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify Your Email</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f9f9f9;
        }
        .container {
            max-width: 600px;
            margin: 20px auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background-color: #10b981;
            color: white;
            padding: 20px;
            text-align: center;
        }
        .content {
            padding: 20px 30px;
        }
        .footer {
            background-color: #f5f5f5;
            padding: 15px;
            text-align: center;
            font-size: 12px;
            color: #666;
        }
        .button {
            display: inline-block;
            background-color: #10b981;
            color: white;
            text-decoration: none;
            padding: 12px 24px;
            border-radius: 4px;
            margin: 20px 0;
            font-weight: 500;
            text-align: center;
        }
        .button:hover {
            background-color: #0d9668;
        }
        .link {
            word-break: break-all;
            color: #10b981;
        }
        .expiry {
            background-color: #f0fdf4;
            border-left: 4px solid #10b981;
            padding: 10px 15px;
            margin: 15px 0;
            font-size: 14px;
        }
        .logo {
            max-width: 150px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="https://cirrussync.me/logo-white.png" alt="CirrusSync Logo" class="logo">
            <h1>Email Verification</h1>
        </div>
        <div class="content">
            <h2>Hello, %s!</h2>
            <p>Thank you for signing up for CirrusSync. To complete your registration, please verify your email address by clicking the button below:</p>

            <div style="text-align: center;">
                <a href="%s" class="button">Verify Email Address</a>
            </div>

            <p>Or copy and paste the following URL into your browser:</p>
            <p class="link">%s</p>

            <div class="expiry">
                <p><strong>Note:</strong> This verification link will expire in %s.</p>
            </div>

            <p>If you didn't sign up for CirrusSync, please ignore this email or contact our support team if you have any concerns.</p>

            <p>Thank you,<br>The CirrusSync Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 CirrusSync. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`, username, verificationURL, verificationURL, expiry)

		// Plain text version
		textTemplate = fmt.Sprintf(`
Hello %s,

Thank you for signing up for CirrusSync! Please verify your email address by clicking the link below:

%s

This link will expire in %s.

If you did not sign up for CirrusSync, please ignore this email.

Best regards,
The CirrusSync Team
`, username, verificationURL, expiry)

	case "password-reset":
		subject = "Password Reset Request - CirrusSync"

		// HTML version
		htmlTemplate = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Password Reset</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f9f9f9;
        }
        .container {
            max-width: 600px;
            margin: 20px auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: a 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background-color: #10b981;
            color: white;
            padding: 20px;
            text-align: center;
        }
        .content {
            padding: 20px 30px;
        }
        .footer {
            background-color: #f5f5f5;
            padding: 15px;
            text-align: center;
            font-size: 12px;
            color: #666;
        }
        .button {
            display: inline-block;
            background-color: #10b981;
            color: white;
            text-decoration: none;
            padding: 12px 24px;
            border-radius: 4px;
            margin: 20px 0;
            font-weight: 500;
            text-align: center;
        }
        .button:hover {
            background-color: #0d9668;
        }
        .link {
            word-break: break-all;
            color: #10b981;
        }
        .expiry {
            background-color: #f0fdf4;
            border-left: 4px solid #10b981;
            padding: 10px 15px;
            margin: 15px 0;
            font-size: 14px;
        }
        .security {
            background-color: #fff8f1;
            border-left: 4px solid #f59e0b;
            padding: 10px 15px;
            margin: 15px 0;
            font-size: 14px;
        }
        .logo {
            max-width: 150px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="https://cirrussync.me/logo-white.png" alt="CirrusSync Logo" class="logo">
            <h1>Password Reset</h1>
        </div>
        <div class="content">
            <h2>Hello, %s!</h2>
            <p>We received a request to reset your password for CirrusSync. To reset your password, please click the button below:</p>

            <div style="text-align: center;">
                <a href="%s" class="button">Reset Password</a>
            </div>

            <p>Or copy and paste the following URL into your browser:</p>
            <p class="link">%s</p>

            <div class="expiry">
                <p><strong>Note:</strong> This password reset link will expire in %s.</p>
            </div>

            <div class="security">
                <p><strong>Security Notice:</strong> If you did not request a password reset, please ignore this email or contact our support team immediately as your account security might be at risk.</p>
            </div>

            <p>Thank you,<br>The CirrusSync Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 CirrusSync. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`, username, verificationURL, verificationURL, expiry)

		// Plain text version
		textTemplate = fmt.Sprintf(`
Hello %s,

We received a request to reset your password for CirrusSync. Please click the link below to reset your password:

%s

This link will expire in %s.

If you did not request a password reset, please ignore this email or contact support if you have concerns.

Best regards,
The CirrusSync Team
`, username, verificationURL, expiry)

	case "2fa":
		subject = "Two-Factor Authentication Setup - CirrusSync"

		// HTML version
		htmlTemplate = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Two-Factor Authentication Setup</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f9f9f9;
        }
        .container {
            max-width: 600px;
            margin: 20px auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background-color: #10b981;
            color: white;
            padding: 20px;
            text-align: center;
        }
        .content {
            padding: 20px 30px;
        }
        .footer {
            background-color: #f5f5f5;
            padding: 15px;
            text-align: center;
            font-size: 12px;
            color: #666;
        }
        .button {
            display: inline-block;
            background-color: #10b981;
            color: white;
            text-decoration: none;
            padding: 12px 24px;
            border-radius: 4px;
            margin: 20px 0;
            font-weight: 500;
            text-align: center;
        }
        .button:hover {
            background-color: #0d9668;
        }
        .link {
            word-break: break-all;
            color: #10b981;
        }
        .expiry {
            background-color: #f0fdf4;
            border-left: 4px solid #10b981;
            padding: 10px 15px;
            margin: 15px 0;
            font-size: 14px;
        }
        .security {
            background-color: #fff8f1;
            border-left: 4px solid #f59e0b;
            padding: 10px 15px;
            margin: 15px 0;
            font-size: 14px;
        }
        .logo {
            max-width: 150px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="https://cirrussync.me/logo-white.png" alt="CirrusSync Logo" class="logo">
            <h1>Two-Factor Authentication</h1>
        </div>
        <div class="content">
            <h2>Hello, %s!</h2>
            <p>We received a request to set up two-factor authentication (2FA) for your CirrusSync account. To continue with the setup process, please click the button below:</p>

            <div style="text-align: center;">
                <a href="%s" class="button">Set Up 2FA</a>
            </div>

            <p>Or copy and paste the following URL into your browser:</p>
            <p class="link">%s</p>

            <div class="expiry">
                <p><strong>Note:</strong> This setup link will expire in %s.</p>
            </div>

            <div class="security">
                <p><strong>Security Notice:</strong> Two-factor authentication adds an extra layer of security to your account. Once set up, you'll need both your password and a verification code to sign in.</p>
                <p>If you did not request to set up 2FA, please ignore this email or contact our support team immediately as someone might be trying to access your account.</p>
            </div>

            <p>Thank you,<br>The CirrusSync Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 CirrusSync. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`, username, verificationURL, verificationURL, expiry)

		// Plain text version
		textTemplate = fmt.Sprintf(`
Hello %s,

We received a request to set up two-factor authentication for your CirrusSync account. Please click the link below to continue:

%s

This link will expire in %s.

If you did not request to set up 2FA, please ignore this email or contact support immediately as someone might be trying to access your account.

Best regards,
The CirrusSync Team
`, username, verificationURL, expiry)

	default:
		// Generic verification email as fallback
		subject = "Email Verification - CirrusSync"

		// HTML version
		htmlTemplate = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Email Verification</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f9f9f9;
        }
        .container {
            max-width: 600px;
            margin: 20px auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background-color: #10b981;
            color: white;
            padding: 20px;
            text-align: center;
        }
        .content {
            padding: 20px 30px;
        }
        .footer {
            background-color: #f5f5f5;
            padding: 15px;
            text-align: center;
            font-size: 12px;
            color: #666;
        }
        .button {
            display: inline-block;
            background-color: #10b981;
            color: white;
            text-decoration: none;
            padding: 12px 24px;
            border-radius: 4px;
            margin: 20px 0;
            font-weight: 500;
            text-align: center;
        }
        .button:hover {
            background-color: #0d9668;
        }
        .link {
            word-break: break-all;
            color: #10b981;
        }
        .expiry {
            background-color: #f0fdf4;
            border-left: 4px solid #10b981;
            padding: 10px 15px;
            margin: 15px 0;
            font-size: 14px;
        }
        .logo {
            max-width: 150px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="https://cirrussync.me/logo-white.png" alt="CirrusSync Logo" class="logo">
            <h1>Email Verification</h1>
        </div>
        <div class="content">
            <h2>Hello, %s!</h2>
            <p>Please verify your email address by clicking the button below:</p>

            <div style="text-align: center;">
                <a href="%s" class="button">Verify Email Address</a>
            </div>

            <p>Or copy and paste the following URL into your browser:</p>
            <p class="link">%s</p>

            <div class="expiry">
                <p><strong>Note:</strong> This verification link will expire in %s.</p>
            </div>

            <p>Thank you,<br>The CirrusSync Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 CirrusSync. All rights reserved.</p>
            <p>This is an automated message, please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>
`, username, verificationURL, verificationURL, expiry)

		// Plain text version
		textTemplate = fmt.Sprintf(`
Hello %s,

Please verify your email address by clicking the link below:

%s

This link will expire in %s.

Thank you,
The CirrusSync Team
`, username, verificationURL, expiry)
	}

	return subject, htmlTemplate, textTemplate
}

// VerifyEmail verifies an email using a token
func (s *Service) VerifyEmail(ctx context.Context, token string) (string, bool) {
	if token == "" {
		return "", false
	}

	// Find token in Redis
	tokenKey := emailTokenPrefix + token
	tokenData, err := s.redisClient.Get(ctx, tokenKey)
	if err != nil || tokenData == "" {
		s.logger.Error("Failed to retrieve token or token not found", "error", err)
		return "", false
	}

	// Parse token data (email:username:intent)
	parts := strings.Split(tokenData, ":")
	if len(parts) < 2 {
		s.logger.Error("Invalid token data format", "tokenData", tokenData)
		return "", false
	}

	email := parts[0]
	// Extract intent if available
	intent := "signup" // Default intent
	if len(parts) >= 3 {
		intent = parts[2]
	}

	println(intent)

	// Mark email as verified using a set
	verifiedKey := emailVerifiedPrefix + email
	_, err = s.redisClient.SAdd(ctx, verifiedKey, "true")
	if err != nil {
		s.logger.Error("Failed to mark email as verified in Redis", "email", email, "error", err)
		return "", false
	}

	// Set expiry on the verified flag
	_, err = s.redisClient.Expire(ctx, verifiedKey, emailVerifiedExpiry)
	if err != nil {
		s.logger.Error("Failed to set expiry on verified flag", "email", email, "error", err)
	}

	// Delete the token
	_, err = s.redisClient.Delete(ctx, tokenKey)
	if err != nil {
		s.logger.Error("Failed to delete token from Redis", "token", token, "error", err)
	}

	return email, true
}

// IsEmailVerified checks if an email has been verified
func (s *Service) IsEmailVerified(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, ErrInvalidEmail
	}

	// Normalize email
	email = NormalizeEmail(email)

	verifiedKey := emailVerifiedPrefix + email
	return s.redisClient.SIsMember(ctx, verifiedKey, "true")
}

// EnableTOTP generates TOTP for a user
func (s *Service) EnableTOTP(ctx context.Context, userID string) (*TOTPData, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	// Check if TOTP is already enabled for the user
	enabled, err := s.IsTOTPEnabled(ctx, userID)
	if err != nil {
		return nil, err
	}
	if enabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	// Use distributed lock to prevent race conditions
	lockName := fmt.Sprintf("totp_setup:%s", userID)
	acquired, err := s.redisClient.AcquireLock(ctx, lockName, 30*time.Second, 3, 100*time.Millisecond)
	if err != nil {
		s.logger.Error("Error acquiring lock for TOTP setup", "userID", userID, "error", err)
		return nil, err
	} else if !acquired {
		s.logger.Warn("TOTP setup already in progress", "userID", userID)
		return nil, ErrTOTPSetupInProgress
	}

	// Release lock when done
	defer func() {
		_, err := s.redisClient.ReleaseLock(ctx, lockName)
		if err != nil {
			s.logger.Error("Failed to release lock", "lock", lockName, "error", err)
		}
	}()

	// Generate a new TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.config.TOTPIssuer,
		AccountName: userID,
		Digits:      s.config.TOTPDigits,
		Period:      s.config.TOTPPeriod,
		Algorithm:   s.config.TOTPAlgorithm,
	})
	if err != nil {
		s.logger.Error("Failed to generate TOTP key", "error", err)
		return nil, err
	}

	// Generate recovery keys
	recoveryKeys, err := s.generateRecoveryKeys(10) // Generate 5 recovery keys
	if err != nil {
		s.logger.Error("Failed to generate recovery keys", "error", err)
		return nil, err
	}

	// Store the TOTP secret in Redis (but don't enable it until verification)
	secretKey := totpSecretPrefix + userID
	err = s.redisClient.Set(ctx, secretKey, key.Secret(), totpSecretExpiry)
	if err != nil {
		s.logger.Error("Failed to store TOTP secret in Redis", "error", err)
		return nil, err
	}

	// Store recovery keys (hashed with argon2)
	for i, recoveryKey := range recoveryKeys {
		// Hash the recovery key
		salt := make([]byte, argon2SaltLength)
		_, err := rand.Read(salt)
		if err != nil {
			s.logger.Error("Failed to generate salt for recovery key", "error", err)
			continue
		}

		hash := argon2.IDKey(
			[]byte(recoveryKey),
			salt,
			argon2Iterations,
			argon2Memory,
			argon2Parallelism,
			argon2KeyLength,
		)

		// Combine salt and hash for storage
		hashData := append(salt, hash...)
		encodedHash := base64.StdEncoding.EncodeToString(hashData)

		// Store in Redis
		recoveryKeyRedisKey := fmt.Sprintf("%s%s:%d", totpRecoveryPrefix, userID, i)
		err = s.redisClient.Set(ctx, recoveryKeyRedisKey, encodedHash, totpSecretExpiry)
		if err != nil {
			s.logger.Error("Failed to store recovery key in Redis", "error", err)
			// Continue anyway, non-critical
		}
	}

	// Create response
	data := &TOTPData{
		Secret:       key.Secret(),
		QRCodeURL:    key.URL(),
		RecoveryKeys: recoveryKeys,
	}

	return data, nil
}

// VerifyTOTP verifies a TOTP code and enables TOTP if valid
func (s *Service) VerifyTOTP(ctx context.Context, userID string, code string) (bool, error) {
	if userID == "" || code == "" {
		return false, ErrInvalidInput
	}

	// Normalize TOTP code
	code = NormalizeTOTPCode(code)

	// Validate TOTP code format
	if !ValidateTOTPCode(code) {
		return false, ErrInvalidTOTPCode
	}

	// Get TOTP secret from Redis
	secretKey := totpSecretPrefix + userID
	secret, err := s.redisClient.Get(ctx, secretKey)
	if err != nil || secret == "" {
		s.logger.Error("Failed to retrieve TOTP secret", "error", err)
		return false, ErrTOTPNotInitialized
	}

	// Verify the code
	valid, err := totp.ValidateCustom(
		code,
		secret,
		time.Now().UTC(),
		totp.ValidateOpts{
			Digits:    s.config.TOTPDigits,
			Period:    s.config.TOTPPeriod,
			Skew:      s.config.TOTPSkew,
			Algorithm: s.config.TOTPAlgorithm,
		},
	)
	if err != nil {
		s.logger.Error("Failed to validate TOTP code", "error", err)
		return false, err
	}

	if !valid {
		return false, nil
	}

	// Use distributed lock for enabling TOTP
	lockName := fmt.Sprintf("totp_enable:%s", userID)
	acquired, err := s.redisClient.AcquireLock(ctx, lockName, 10*time.Second, 3, 100*time.Millisecond)
	if err != nil {
		s.logger.Error("Error acquiring lock for TOTP enabling", "userID", userID, "error", err)
	} else if !acquired {
		s.logger.Warn("TOTP enabling already in progress", "userID", userID)
		return true, nil // Still return valid since the code was correct
	}

	// Release lock when done
	defer func() {
		_, err := s.redisClient.ReleaseLock(ctx, lockName)
		if err != nil {
			s.logger.Error("Failed to release lock", "lock", lockName, "error", err)
		}
	}()

	// If code is valid, enable TOTP for the user using a set
	enabledKey := totpEnabledPrefix + userID
	_, err = s.redisClient.SAdd(ctx, enabledKey, "true")
	if err != nil {
		s.logger.Error("Failed to enable TOTP in Redis", "error", err)
		return false, err
	}

	// Set expiry on the enabled flag (never expire)
	_, err = s.redisClient.Expire(ctx, enabledKey, totpSecretExpiry)
	if err != nil {
		s.logger.Error("Failed to set expiry on TOTP enabled flag", "userID", userID, "error", err)
	}

	s.logger.Info("TOTP enabled successfully", "userID", userID)
	return true, nil
}

// ValidateTOTPCode validates a TOTP code
func (s *Service) ValidateTOTPCode(ctx context.Context, userID string, code string) (bool, error) {
	if userID == "" || code == "" {
		return false, ErrInvalidInput
	}

	// Normalize TOTP code
	code = NormalizeTOTPCode(code)

	// Check if TOTP is enabled for the user
	enabled, err := s.IsTOTPEnabled(ctx, userID)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, ErrTOTPNotEnabled
	}

	// Check if it's a recovery key first (recovery keys take priority)
	isRecoveryKey, err := s.checkRecoveryKey(ctx, userID, code)
	if err != nil {
		s.logger.Error("Failed to check recovery key", "error", err)
		// Continue with normal validation if there's an error checking recovery keys
	} else if isRecoveryKey {
		// Remove the used recovery key
		err = s.removeRecoveryKey(ctx, userID, code)
		if err != nil {
			s.logger.Error("Failed to remove recovery key", "error", err)
		}
		return true, nil
	}

	// Get TOTP secret from Redis
	secretKey := totpSecretPrefix + userID
	secret, err := s.redisClient.Get(ctx, secretKey)
	if err != nil || secret == "" {
		s.logger.Error("Failed to retrieve TOTP secret", "error", err)
		return false, ErrTOTPNotInitialized
	}

	// Verify the code
	valid, err := totp.ValidateCustom(
		code,
		secret,
		time.Now().UTC(),
		totp.ValidateOpts{
			Digits:    s.config.TOTPDigits,
			Period:    s.config.TOTPPeriod,
			Skew:      s.config.TOTPSkew,
			Algorithm: s.config.TOTPAlgorithm,
		},
	)
	if err != nil {
		s.logger.Error("Failed to validate TOTP code", "error", err)
		return false, err
	}

	return valid, nil
}

// IsTOTPEnabled checks if TOTP is enabled for a user
func (s *Service) IsTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, ErrInvalidInput
	}

	enabledKey := totpEnabledPrefix + userID
	return s.redisClient.SIsMember(ctx, enabledKey, "true")
}

// DisableTOTP disables TOTP for a user
func (s *Service) DisableTOTP(ctx context.Context, userID string, confirmationCode string) error {
	if userID == "" {
		return ErrInvalidInput
	}

	// Optional confirmation with TOTP code
	if confirmationCode != "" {
		valid, err := s.ValidateTOTPCode(ctx, userID, confirmationCode)
		if err != nil {
			return err
		}
		if !valid {
			return ErrInvalidTOTPCode
		}
	}

	// Use distributed lock for disabling TOTP
	lockName := fmt.Sprintf("totp_disable:%s", userID)
	acquired, err := s.redisClient.AcquireLock(ctx, lockName, 10*time.Second, 3, 100*time.Millisecond)
	if err != nil {
		s.logger.Error("Error acquiring lock for TOTP disabling", "userID", userID, "error", err)
		return err
	} else if !acquired {
		s.logger.Warn("TOTP disabling already in progress", "userID", userID)
		return ErrTOTPOperationInProgress
	}

	// Release lock when done
	defer func() {
		_, err := s.redisClient.ReleaseLock(ctx, lockName)
		if err != nil {
			s.logger.Error("Failed to release lock", "lock", lockName, "error", err)
		}
	}()

	// Check if TOTP is actually enabled
	enabled, err := s.IsTOTPEnabled(ctx, userID)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrTOTPNotEnabled
	}

	// Delete TOTP enablement and secrets
	enabledKey := totpEnabledPrefix + userID
	secretKey := totpSecretPrefix + userID

	// Delete keys in Redis
	keys := []string{enabledKey, secretKey}
	_, err = s.redisClient.DeleteMany(ctx, keys...)
	if err != nil {
		s.logger.Error("Failed to delete TOTP keys", "error", err)
		return err
	}

	// Delete recovery keys - find all recovery keys for this user
	for i := 0; i < 10; i++ { // Assume max 10 recovery keys
		recoveryKeyRedisKey := fmt.Sprintf("%s%s:%d", totpRecoveryPrefix, userID, i)
		// Don't care about errors here - best effort cleanup
		s.redisClient.Delete(ctx, recoveryKeyRedisKey)
	}

	s.logger.Info("TOTP disabled successfully", "userID", userID)
	return nil
}

// checkRecoveryKey checks if the provided code is a valid recovery key
func (s *Service) checkRecoveryKey(ctx context.Context, userID, code string) (bool, error) {
	if !strings.Contains(code, "-") {
		return false, nil // Not a recovery key format
	}

	// Strip hyphens for comparison
	normalizedCode := strings.ReplaceAll(code, "-", "")

	// Check all possible recovery keys
	for i := 0; i < 10; i++ { // Assume max 10 recovery keys
		recoveryKeyRedisKey := fmt.Sprintf("%s%s:%d", totpRecoveryPrefix, userID, i)
		storedHash, err := s.redisClient.Get(ctx, recoveryKeyRedisKey)
		if err != nil || storedHash == "" {
			continue // Key doesn't exist or error, try next one
		}

		// Decode the stored hash to get salt and hash
		hashData, err := base64.StdEncoding.DecodeString(storedHash)
		if err != nil || len(hashData) < argon2SaltLength+argon2KeyLength {
			s.logger.Error("Invalid hash data for recovery key", "key", recoveryKeyRedisKey)
			continue
		}

		// Extract salt and hash
		salt := hashData[:argon2SaltLength]
		storedHashPart := hashData[argon2SaltLength:]

		// Hash the provided code
		computedHash := argon2.IDKey(
			[]byte(normalizedCode),
			salt,
			argon2Iterations,
			argon2Memory,
			argon2Parallelism,
			argon2KeyLength,
		)

		// Compare the hashes (constant time comparison)
		if subtle.ConstantTimeCompare(computedHash, storedHashPart) == 1 {
			return true, nil
		}
	}

	return false, nil
}

// removeRecoveryKey removes a used recovery key
func (s *Service) removeRecoveryKey(ctx context.Context, userID, code string) error {
	if !strings.Contains(code, "-") {
		return nil // Not a recovery key format
	}

	// Strip hyphens for comparison
	normalizedCode := strings.ReplaceAll(code, "-", "")

	// Find and remove the used recovery key
	for i := 0; i < 10; i++ { // Assume max 10 recovery keys
		recoveryKeyRedisKey := fmt.Sprintf("%s%s:%d", totpRecoveryPrefix, userID, i)
		storedHash, err := s.redisClient.Get(ctx, recoveryKeyRedisKey)
		if err != nil || storedHash == "" {
			continue // Key doesn't exist or error, try next one
		}

		// Decode the stored hash to get salt and hash
		hashData, err := base64.StdEncoding.DecodeString(storedHash)
		if err != nil || len(hashData) < argon2SaltLength+argon2KeyLength {
			continue
		}

		// Extract salt and hash
		salt := hashData[:argon2SaltLength]
		storedHashPart := hashData[argon2SaltLength:]

		// Hash the provided code
		computedHash := argon2.IDKey(
			[]byte(normalizedCode),
			salt,
			argon2Iterations,
			argon2Memory,
			argon2Parallelism,
			argon2KeyLength,
		)

		// Compare the hashes
		if subtle.ConstantTimeCompare(computedHash, storedHashPart) == 1 {
			// Delete the recovery key
			_, err := s.redisClient.Delete(ctx, recoveryKeyRedisKey)
			if err != nil {
				s.logger.Error("Failed to delete recovery key", "key", recoveryKeyRedisKey, "error", err)
				return err
			}
			return nil
		}
	}

	return nil
}

// generateRecoveryKeys generates a set of recovery keys
func (s *Service) generateRecoveryKeys(count int) ([]string, error) {
	keys := make([]string, count)

	for i := 0; i < count; i++ {
		// Generate 10 bytes of random data (80 bits of entropy)
		b := make([]byte, 10)
		_, err := rand.Read(b)
		if err != nil {
			return nil, err
		}

		// Convert to base32 and format nicely with hyphens
		encoded := strings.ToUpper(base32.StdEncoding.EncodeToString(b))
		encoded = encoded[:16] // Trim any padding

		// Format as XXXX-XXXX-XXXX-XXXX
		formatted := fmt.Sprintf("%s-%s-%s-%s",
			encoded[0:4], encoded[4:8], encoded[8:12], encoded[12:16])

		keys[i] = formatted
	}

	return keys, nil
}

// sendEmailFast sends an email using the SMTP connection pool
func (s *Service) sendEmailFast(to []string, subject, htmlBody, textBody string) error {
	// Get a client from the pool
	client, err := smtpPool.getClient()
	if err != nil {
		return &EmailSendError{
			Email: to[0],
			Err:   err,
		}
	}
	defer smtpPool.releaseClient(client)

	// Create a MIME message with multipart/alternative
	boundary := "==CirrusSyncBoundary=="

	// Compose email message with explicit From header and multipart content
	message := fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=\"%s\"\r\n"+
		"\r\n"+
		"--%s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"Content-Transfer-Encoding: 7bit\r\n"+
		"\r\n"+
		"%s\r\n"+
		"\r\n"+
		"--%s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Content-Transfer-Encoding: 7bit\r\n"+
		"\r\n"+
		"%s\r\n"+
		"\r\n"+
		"--%s--\r\n",
		to[0],
		s.config.FromEmail,
		subject,
		boundary,
		boundary,
		textBody,
		boundary,
		htmlBody,
		boundary)

	client.mu.Lock()
	defer client.mu.Unlock()

	// Reset the connection for a new message
	if err := client.client.Reset(); err != nil {
		// If reset fails, create a new connection
		newClient, createErr := smtpPool.createClient()
		if createErr != nil {
			return &EmailSendError{
				Email: to[0],
				Err:   createErr,
			}
		}
		// Replace the old client
		client = newClient
	}

	// Set the sender and recipients
	if err := client.client.Mail(s.config.FromEmail); err != nil {
		return &EmailSendError{
			Email: to[0],
			Err:   fmt.Errorf("failed to set sender: %w", err),
		}
	}

	for _, addr := range to {
		if err := client.client.Rcpt(addr); err != nil {
			return &EmailSendError{
				Email: addr,
				Err:   fmt.Errorf("failed to set recipient: %w", err),
			}
		}
	}

	// Send the email body
	w, err := client.client.Data()
	if err != nil {
		return &EmailSendError{
			Email: to[0],
			Err:   fmt.Errorf("failed to open data writer: %w", err),
		}
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return &EmailSendError{
			Email: to[0],
			Err:   fmt.Errorf("failed to write email body: %w", err),
		}
	}

	err = w.Close()
	if err != nil {
		return &EmailSendError{
			Email: to[0],
			Err:   fmt.Errorf("failed to close data writer: %w", err),
		}
	}

	return nil
}

// sendEmail is a fallback for when we need to ensure delivery
func (s *Service) sendEmail(to []string, subject, htmlBody, textBody string) error {
	// Create a MIME message with multipart/alternative
	boundary := "==CirrusSyncBoundary=="

	// Compose email message with explicit From header and multipart content
	message := fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=\"%s\"\r\n"+
		"\r\n"+
		"--%s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"Content-Transfer-Encoding: 7bit\r\n"+
		"\r\n"+
		"%s\r\n"+
		"\r\n"+
		"--%s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Content-Transfer-Encoding: 7bit\r\n"+
		"\r\n"+
		"%s\r\n"+
		"\r\n"+
		"--%s--\r\n",
		to[0],
		s.config.FromEmail,
		subject,
		boundary,
		boundary,
		textBody,
		boundary,
		htmlBody,
		boundary)

	// SMTP authentication
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
	}

	// Connect to the server
	smtpAddr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Create the SMTP client
	client, err := smtp.Dial(smtpAddr)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Set the sender and recipients
	if err = client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send the email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Close connection
	return client.Quit()
}

// formatTimeRemaining formats a duration in a user-friendly way
func (s *Service) formatTimeRemaining(d time.Duration) string {
	if d <= 0 {
		return "0 seconds"
	}

	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := []string{}

	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}

	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
	}

	if seconds > 0 || len(parts) == 0 {
		if seconds == 1 {
			parts = append(parts, "1 second")
		} else {
			parts = append(parts, fmt.Sprintf("%d seconds", seconds))
		}
	}

	return strings.Join(parts, " ")
}

// formatDuration formats a duration in a user-friendly way
func (s *Service) formatDuration(d time.Duration) string {
	if d.Hours() >= 24 {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "24 hours"
		}
		return fmt.Sprintf("%d days", days)
	}

	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour"
	} else if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}

	minutes := int(d.Minutes())
	if minutes == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}
