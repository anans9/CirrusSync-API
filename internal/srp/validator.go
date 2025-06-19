// internal/srp/validators.go
package srp

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"cirrussync-api/pkg/redis"
)

// validateEmail checks if an email is valid and normalizes it
func validateEmail(email string) (string, error) {
	if email == "" {
		return "", ErrInvalidInput
	}

	// Normalize email
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	// Basic validation for email format
	if !strings.Contains(normalizedEmail, "@") || len(normalizedEmail) < 5 {
		return "", ErrInvalidInput
	}

	return normalizedEmail, nil
}

// validateClientPublic validates the client's public key
func validateClientPublic(clientPublic string) (*big.Int, error) {
	if clientPublic == "" {
		return nil, ErrInvalidInput
	}

	// Parse client public key
	A, success := new(big.Int).SetString(clientPublic, 16)
	if !success {
		return nil, ErrInvalidClientPublic
	}

	// Ensure A is not zero and less than N
	if A.Cmp(big.NewInt(0)) == 0 || A.Cmp(N) >= 0 {
		return nil, ErrInvalidClientPublic
	}

	return A, nil
}

// validateSessionID validates the session ID format
func validateSessionID(sessionID string) error {
	if sessionID == "" {
		return ErrInvalidInput
	}

	// Check if the session ID has a valid format (hex string)
	if len(sessionID) < 32 {
		return ErrInvalidSession
	}

	return nil
}

// validateClientProof validates the client proof format
func validateClientProof(clientProof string) ([]byte, error) {
	if clientProof == "" {
		return nil, ErrInvalidInput
	}

	// Check if the client proof has a valid format (hex string)
	if !isHexString(clientProof) {
		return nil, ErrInvalidClientProof
	}

	// Convert to bytes
	bytes, err := hex.DecodeString(clientProof)
	if err != nil {
		return nil, ErrInvalidClientProof
	}

	return bytes, nil
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// checkRateLimiting checks if the authentication request should be rate limited
func checkRateLimiting(ctx context.Context, redisClient *redis.Client, email, ipAddress string) error {
	// Check rate limiting for email
	emailKey := fmt.Sprintf("srp:failed:%s", email)
	emailAttemptsStr, err := redisClient.Get(ctx, emailKey)

	var emailAttempts int
	if err == nil && emailAttemptsStr != "" {
		fmt.Sscanf(emailAttemptsStr, "%d", &emailAttempts)
		if emailAttempts >= 3 {
			return ErrRateLimited
		}
	}

	// Check rate limiting for IP address
	ipKey := fmt.Sprintf("srp:failed:ip:%s", ipAddress)
	ipAttemptsStr, err := redisClient.Get(ctx, ipKey)

	var ipAttempts int
	if err == nil && ipAttemptsStr != "" {
		fmt.Sscanf(ipAttemptsStr, "%d", &ipAttempts)
		if ipAttempts >= 5 {
			return ErrRateLimited
		}
	}

	// Check global rate limiting (prevent distributed attacks)
	globalKey := "srp:global:request_count"
	countStr, err := redisClient.Get(ctx, globalKey)
	var count int64 = 0

	if err == nil && countStr != "" {
		fmt.Sscanf(countStr, "%d", &count)
	}

	count++

	// Store updated count
	redisClient.Set(ctx, globalKey, fmt.Sprintf("%d", count), 1*time.Minute)

	// If more than 100 requests per minute globally, start more aggressive rate limiting
	if count > 100 {
		// Random rate limiting to prevent timing attacks
		if count%(1+count/200) == 0 {
			time.Sleep(time.Duration(100+int(time.Now().UnixNano()%300)) * time.Millisecond)
		}
	}

	return nil
}

// GetClientIPFromRequest extracts the client's real IP address with Cloudflare support
func GetClientIPFromRequest(r *http.Request) string {
	// First check for Cloudflare-specific headers
	// CF-Connecting-IP is the most reliable when using Cloudflare
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		if net.ParseIP(cfIP) != nil {
			return cfIP
		}
	}

	// Check if request is coming from Cloudflare
	if r.Header.Get("CF-Ray") != "" {
		// True-Client-IP is set by Cloudflare
		if trueClientIP := r.Header.Get("True-Client-IP"); trueClientIP != "" {
			if net.ParseIP(trueClientIP) != nil {
				return trueClientIP
			}
		}

		// X-Forwarded-For from Cloudflare will have client's IP as the first entry
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			ip := strings.TrimSpace(ips[0])

			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	} else {
		// Standard proxy headers
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			ip := strings.TrimSpace(ips[0])

			if net.ParseIP(ip) != nil {
				return ip
			}
		}

		// Check for X-Real-IP header
		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			if net.ParseIP(xrip) != nil {
				return xrip
			}
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
