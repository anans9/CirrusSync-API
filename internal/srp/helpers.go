package srp

import (
	"cirrussync-api/internal/utils"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/sirupsen/logrus"
)

// Cryptographic helper functions starts

// calculateK computes k = H(N, g) as per SRP-6a
func calculateK(N, g *big.Int) *big.Int {
	h := sha256.New()

	// Pad N to byte array
	nBytes := padBigInt(N, (N.BitLen()+7)/8)
	h.Write(nBytes)

	// Pad g to same length as N
	gBytes := padBigInt(g, len(nBytes))
	h.Write(gBytes)

	return new(big.Int).SetBytes(h.Sum(nil))
}

// padBigInt pads a big.Int to a specific byte length
func padBigInt(n *big.Int, length int) []byte {
	bytes := n.Bytes()
	if len(bytes) >= length {
		return bytes
	}

	padded := make([]byte, length)
	copy(padded[length-len(bytes):], bytes)
	return padded
}

// generateRandomBigInt generates a cryptographically secure random bigint in [1, N-1]
func generateRandomBigInt(max *big.Int) (*big.Int, error) {
	// Ensure value is in [1, max-1]
	min := big.NewInt(1)
	diff := new(big.Int).Sub(max, min)

	n, err := rand.Int(rand.Reader, diff)
	if err != nil {
		return nil, err
	}

	return n.Add(n, min), nil
}

// calculateU computes u = H(A, B)
func calculateU(A, B *big.Int) *big.Int {
	h := sha256.New()

	// Ensure A and B are properly padded
	aBytes := padBigInt(A, (A.BitLen()+7)/8)
	bBytes := padBigInt(B, (B.BitLen()+7)/8)

	h.Write(aBytes)
	h.Write(bBytes)

	return new(big.Int).SetBytes(h.Sum(nil))
}

// calculateClientProof calculates M1 = H(A | B | K)
func calculateClientProof(A, B *big.Int, K []byte) []byte {
	h := sha256.New()

	// Ensure proper padding
	aBytes := padBigInt(A, (A.BitLen()+7)/8)
	bBytes := padBigInt(B, (B.BitLen()+7)/8)

	h.Write(aBytes)
	h.Write(bBytes)
	h.Write(K)

	return h.Sum(nil)
}

// calculateServerProof calculates M2 = H(A | M1 | K)
func calculateServerProof(A *big.Int, M1, K []byte) []byte {
	h := sha256.New()

	// Ensure proper padding
	aBytes := padBigInt(A, (A.BitLen()+7)/8)

	h.Write(aBytes)
	h.Write(M1)
	h.Write(K)

	return h.Sum(nil)
}

// generateFakeSalt generates a fake salt to use for non-existent users
func generateFakeSalt() string {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		// If random generation fails, use a fixed salt
		return "71e4f9548c48f20299f3c3f33e04a273f89bce673b39729c6c9535343cf1fdef"
	}
	return hex.EncodeToString(salt)
}

// SRP helper methods

// storeSession stores an SRP session in Redis
func (s *Service) storeSession(ctx context.Context, sessionID string, session *Session) error {
	// Redis key for session
	key := fmt.Sprintf("srp:session:%s", sessionID)

	// Calculate expiration time as duration
	ttl := session.ExpiresAt.Sub(time.Now())

	// Store in Redis with TTL
	return s.redisClient.SetJSON(ctx, key, session, ttl)
}

// getSession retrieves an SRP session from Redis
func (s *Service) getSession(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("srp:session:%s", sessionID)

	// Get from Redis
	var session Session
	err := s.redisClient.GetJSON(ctx, key, &session)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		s.redisClient.Delete(ctx, key)
		return nil, ErrInvalidSession
	}

	return &session, nil
}

// deleteSession removes an SRP session from Redis
func (s *Service) deleteSession(ctx context.Context, sessionID string) {
	key := fmt.Sprintf("srp:session:%s", sessionID)
	s.redisClient.Delete(ctx, key)
}

// generateFakeResponse generates a fake response for non-existent users
func (s *Service) generateFakeResponse() (*InitResponse, error) {
	// Generate a fake salt
	fakeSalt := generateFakeSalt()

	// Generate a random server ephemeral value
	b, err := generateRandomBigInt(N)
	if err != nil {
		return nil, ErrServerError
	}

	// Create a fake server public key B = g^b % N
	B := new(big.Int).Exp(g, b, N)

	return &InitResponse{
		SessionID:    utils.GenerateLinkID(),
		Salt:         fakeSalt,
		ServerPublic: B.Text(16),
	}, nil
}

// recordFailedAttempt records a failed authentication attempt
func (s *Service) recordFailedAttempt(ctx context.Context, email, ipAddress string) {
	// Increment and set expiry for email counter
	emailKey := fmt.Sprintf("srp:failed:%s", email)
	emailAttemptsStr, err := s.redisClient.Get(ctx, emailKey)

	var attempts int64 = 0
	if err == nil && emailAttemptsStr != "" {
		fmt.Sscanf(emailAttemptsStr, "%d", &attempts)
	}

	attempts++

	// Set TTL based on number of attempts
	var ttl time.Duration = 10 * time.Minute
	if attempts >= 10 {
		ttl = 24 * time.Hour
		// Log suspicious activity
		s.logger.WithFields(logrus.Fields{
			"email": email,
			"ip":    ipAddress,
		}).Warn("Possible brute force attack detected")
	} else if attempts >= 5 {
		ttl = 2 * time.Hour
	} else if attempts >= 3 {
		ttl = 30 * time.Minute
	}

	s.redisClient.Set(ctx, emailKey, fmt.Sprintf("%d", attempts), ttl)

	// Increment and set expiry for IP counter
	ipKey := fmt.Sprintf("srp:failed:ip:%s", ipAddress)
	ipAttemptsStr, err := s.redisClient.Get(ctx, ipKey)

	var ipAttempts int64 = 0
	if err == nil && ipAttemptsStr != "" {
		fmt.Sscanf(ipAttemptsStr, "%d", &ipAttempts)
	}

	ipAttempts++

	// Set TTL based on number of attempts
	var ipTTL time.Duration = 10 * time.Minute
	if ipAttempts >= 10 {
		ipTTL = 24 * time.Hour
		// Log suspicious IP
		s.logger.WithField("ip", ipAddress).Warn("Possible IP-based attack detected")
	} else if ipAttempts >= 5 {
		ipTTL = 1 * time.Hour
	}

	s.redisClient.Set(ctx, ipKey, fmt.Sprintf("%d", ipAttempts), ipTTL)
}

// resetFailedAttempts resets the failed attempt counters after successful authentication
func (s *Service) resetFailedAttempts(ctx context.Context, email, ipAddress string) {
	// Reset email counter
	emailKey := fmt.Sprintf("srp:failed:%s", email)
	s.redisClient.Delete(ctx, emailKey)

	// Don't reset IP counter completely to prevent distributed attacks
	// Instead, decrement it to allow legitimate future attempts
	ipKey := fmt.Sprintf("srp:failed:ip:%s", ipAddress)
	ipAttemptsStr, err := s.redisClient.Get(ctx, ipKey)

	var ipAttempts int64 = 0
	if err == nil && ipAttemptsStr != "" {
		fmt.Sscanf(ipAttemptsStr, "%d", &ipAttempts)

		// Decrement by 2 to reward successful login
		if ipAttempts > 2 {
			ipAttempts -= 2
			s.redisClient.Set(ctx, ipKey, fmt.Sprintf("%d", ipAttempts), 10*time.Minute)
		} else {
			s.redisClient.Delete(ctx, ipKey)
		}
	}
}
