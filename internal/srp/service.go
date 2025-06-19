// internal/srp/service.go
package srp

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"math/big"
	"time"

	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/redis"

	"github.com/sirupsen/logrus"
)

// SRP-6a parameters (2048-bit)
var (
	N *big.Int // Safe prime
	g *big.Int // Generator
	k *big.Int // Multiplier parameter (k = H(N, g))
)

// Initialize SRP parameters
func init() {
	// Initialize N (safe prime) from the hex string
	N = new(big.Int)
	N.SetString("AC6BDB41324A9A9BF166DE5E1389582FAF72B6651987EE07FC3192943DB56050A37329CBB4A099ED8193E0757767A13DD52312AB4B03310DCD7F48A9DA04FD50E8083969EDB767B0CF6095179A163AB3661A05FBD5FAAAE82918A9962F0B93B855F97993EC975EEAA80D740ADBF4FF747359D041D5C33EA71D281E446B14773BCA97B43A23FB801676BD207A436C6481F1D2B9078717461A5B9D32E688F87748544523B524B0D57D5EA77A2775D2ECFA032CFBDBF52FB3786160279004E57AE6AF874E7303CE53299CCC041C7BC308D82A5698F3A8D0C38271AE35F8E9DBFBB694B5C803D89F7AE435DE236D525F54759B65E372FCD68EF20FA7111F9E4AFF73", 16)

	// Initialize g (generator)
	g = big.NewInt(2)

	// Calculate k = H(N, g)
	k = calculateK(N, g)
}

// Session represents an in-progress SRP authentication session
type Session struct {
	UserID        string    `json:"user_id"`
	Email         string    `json:"email"`
	Salt          string    `json:"salt"`
	Verifier      string    `json:"verifier"`
	ClientPublic  string    `json:"client_public"`
	ServerPrivate string    `json:"server_private"`
	ServerPublic  string    `json:"server_public"`
	SessionKey    string    `json:"session_key"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
}

// InitResponse is returned from the initialization phase
type InitResponse struct {
	SessionID    string `json:"session_id"`
	Salt         string `json:"salt"`
	ServerPublic string `json:"server_public"`
}

// VerifyResponse contains the server proof for verification
type VerifyResponse struct {
	ServerProof string `json:"server_proof"`
}

// Service handles SRP authentication business logic
type Service struct {
	repo        Repository
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewService creates a new SRP service
func NewService(repo Repository, redisClient *redis.Client, logger *logger.Logger) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
		logger:      logger,
	}
}

// InitAuthentication initializes the SRP authentication process
func (s *Service) InitAuthentication(ctx context.Context, email, clientPublic, ipAddress, userAgent string) (*InitResponse, error) {
	// Validate and normalize email
	normalizedEmail, err := validateEmail(email)
	if err != nil {
		return nil, err
	}

	// Check rate limiting
	if err := checkRateLimiting(ctx, s.redisClient, normalizedEmail, ipAddress); err != nil {
		return nil, err
	}

	// Validate client public key
	A, err := validateClientPublic(clientPublic)
	if err != nil {
		s.recordFailedAttempt(ctx, normalizedEmail, ipAddress)
		return nil, err
	}

	// Get user SRP data
	userSRP, err := s.repo.GetUserSRPByEmail(normalizedEmail)
	if err != nil {
		// Don't reveal user existence, return a fake response
		s.recordFailedAttempt(ctx, normalizedEmail, ipAddress)
		return s.generateFakeResponse()
	}

	// Parse verifier from DB
	v := new(big.Int)
	v.SetString(userSRP.Verifier, 16)

	// Generate random server private key
	b, err := generateRandomBigInt(N)
	if err != nil {
		s.logger.Error("Failed to generate server private key", err)
		return nil, ErrServerError
	}

	// Calculate server public key: B = kv + g^b
	kv := new(big.Int).Mul(k, v)
	gb := new(big.Int).Exp(g, b, N)
	B := new(big.Int).Add(kv, gb)
	B.Mod(B, N)

	// Calculate u = H(A, B)
	u := calculateU(A, B)

	// Calculate S = (A * v^u)^b % N
	vu := new(big.Int).Exp(v, u, N)
	Avu := new(big.Int).Mul(A, vu)
	Avu.Mod(Avu, N)
	S := new(big.Int).Exp(Avu, b, N)

	// Calculate K = H(S)
	SBytes := padBigInt(S, (S.BitLen()+7)/8)
	K := sha256.Sum256(SBytes)

	// Create session
	sessionID := utils.GenerateLinkID()
	session := &Session{
		UserID:        userSRP.UserID,
		Email:         normalizedEmail,
		Salt:          userSRP.Salt,
		Verifier:      userSRP.Verifier,
		ClientPublic:  A.Text(16),
		ServerPrivate: b.Text(16),
		ServerPublic:  B.Text(16),
		SessionKey:    hex.EncodeToString(K[:]),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
	}

	// Store session in Redis
	if err := s.storeSession(ctx, sessionID, session); err != nil {
		s.logger.Error("Failed to store SRP session", err)
		return nil, ErrServerError
	}

	// Return session ID, salt, and server public key
	return &InitResponse{
		SessionID:    sessionID,
		Salt:         userSRP.Salt,
		ServerPublic: B.Text(16),
	}, nil
}

// VerifyAuthentication verifies the client proof and generates a server proof
func (s *Service) VerifyAuthentication(ctx context.Context, sessionID, clientProof, ipAddress string) (*VerifyResponse, *models.UserSRP, error) {
	// Validate session ID
	if err := validateSessionID(sessionID); err != nil {
		return nil, nil, err
	}

	// Validate client proof
	clientProofBytes, err := validateClientProof(clientProof)
	if err != nil {
		return nil, nil, err
	}

	// Get session from Redis
	session, err := s.getSession(ctx, sessionID)
	if err != nil {
		return nil, nil, ErrInvalidSession
	}

	// Check IP address to prevent session hijacking
	if session.IPAddress != ipAddress {
		// Using the logger.WithFields method
		s.logger.WithFields(logrus.Fields{
			"session_ip": session.IPAddress,
			"request_ip": ipAddress,
			"email":      session.Email,
		}).Warn("IP address mismatch during SRP verification")

		s.recordFailedAttempt(ctx, session.Email, ipAddress)
		return nil, nil, ErrInvalidSession
	}

	// Parse stored values
	A, success := new(big.Int).SetString(session.ClientPublic, 16)
	if !success {
		s.recordFailedAttempt(ctx, session.Email, ipAddress)
		return nil, nil, ErrInvalidClientPublic
	}

	B, success := new(big.Int).SetString(session.ServerPublic, 16)
	if !success {
		return nil, nil, ErrInvalidServerPublic
	}

	K, err := hex.DecodeString(session.SessionKey)
	if err != nil {
		return nil, nil, ErrServerError
	}

	// Calculate expected client proof: M1 = H(A | B | K)
	expectedM1 := calculateClientProof(A, B, K)

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(expectedM1, clientProofBytes) != 1 {
		s.recordFailedAttempt(ctx, session.Email, ipAddress)
		return nil, nil, ErrInvalidClientProof
	}

	// Get user SRP data
	userSRP, err := s.repo.GetUserSRPByEmail(session.Email)
	if err != nil {
		s.logger.WithField("email", session.Email).Error("Failed to get user SRP data during verification", err)
		return nil, nil, ErrUserNotFound
	}

	// Calculate server proof: M2 = H(A | M1 | K)
	serverProof := calculateServerProof(A, clientProofBytes, K)

	// Authentication successful - reset failed attempts
	s.resetFailedAttempts(ctx, session.Email, ipAddress)

	// Delete the session
	s.deleteSession(ctx, sessionID)

	// Return server proof
	return &VerifyResponse{
		ServerProof: hex.EncodeToString(serverProof),
	}, userSRP, nil
}

// RegisteSRPCredentials registers new SRP credentials for a user
func (s *Service) RegisterSRPCredentials(ctx context.Context, userID, email, salt, verifier string) error {
	// Validate and normalize email
	normalizedEmail, err := validateEmail(email)
	if err != nil {
		return err
	}

	// Validate salt and verifier
	if salt == "" || verifier == "" {
		return ErrInvalidInput
	}

	// Validate that the verifier is a valid hex string
	_, err = hex.DecodeString(verifier)
	if err != nil {
		return ErrInvalidInput
	}

	// Check if SRP credentials already exist
	existingSRP, err := s.repo.GetUserSRP(userID)

	now := time.Now().Unix()

	if err == nil && existingSRP != nil {
		// Update existing credentials
		existingSRP.Salt = salt
		existingSRP.Verifier = verifier
		existingSRP.Version++
		existingSRP.ModifiedAt = now
		existingSRP.Active = true

		err = s.repo.UpdateUserSRP(existingSRP)
		if err != nil {
			s.logger.Error("Failed to update SRP credentials", err)
			return err
		}

		return nil
	}

	// Create new SRP credentials
	newSRP := &models.UserSRP{
		ID:         utils.GenerateLinkID(),
		UserID:     userID,
		Email:      normalizedEmail,
		Salt:       salt,
		Verifier:   verifier,
		Version:    1,
		CreatedAt:  now,
		ModifiedAt: now,
		Active:     true,
	}

	err = s.repo.SaveUserSRP(newSRP)
	if err != nil {
		s.logger.Error("Failed to save SRP credentials", err)
		return err
	}

	return nil
}

// Get User SRP Credentials

func (s *Service) GetUserSRPByID(userID string) (*models.UserSRP, error) {
	// Fetch user's SRP parameters from database
	userSRP, err := s.repo.GetUserSRP(userID)
	if err != nil {
		return nil, err
	}

	return userSRP, nil
}
