package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"os"
	"sync/atomic"
	"time"
)

// Counter for sequential uniqueness
var sequenceCounter uint64 = 0

// GenerateID creates a cryptographically secure unique ID
// Uses only cryptographic primitives with timestamp and counter for guaranteed uniqueness
func GenerateID() string {
	// Create a buffer for our combined entropy sources
	buf := make([]byte, 0, 128)

	// 1. Add high-precision timestamp (8 bytes)
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(time.Now().UnixNano()))
	buf = append(buf, timeBytes...)

	// 2. Add process ID (4 bytes)
	pidBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(pidBytes, uint32(os.Getpid()))
	buf = append(buf, pidBytes...)

	// 3. Add atomic counter for sequential uniqueness (8 bytes)
	counterBytes := make([]byte, 8)
	counter := atomic.AddUint64(&sequenceCounter, 1)
	binary.BigEndian.PutUint64(counterBytes, counter)
	buf = append(buf, counterBytes...)

	// 4. Add cryptographically secure random bytes (32 bytes)
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to something deterministic but still unique if rand fails
		h := sha256.New()
		h.Write(buf)
		h.Write([]byte(time.Now().String()))
		randomBytes = h.Sum(nil)
	}
	buf = append(buf, randomBytes...)

	// 5. Hash everything with SHA-256 for fixed size output
	hash := sha256.Sum256(buf)

	// 6. Encode to URL-safe base64
	encoded := base64.URLEncoding.EncodeToString(hash[:])

	// Remove padding and return
	return encoded[:43] // Without padding
}

// GenerateShortID creates a shorter ID (good for visible IDs but still secure)
func GenerateShortID() string {
	// Create buffer with entropy
	buf := make([]byte, 0, 64)

	// Add timestamp
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(time.Now().UnixNano()))
	buf = append(buf, timeBytes...)

	// Add counter
	counterBytes := make([]byte, 8)
	counter := atomic.AddUint64(&sequenceCounter, 1)
	binary.BigEndian.PutUint64(counterBytes, counter)
	buf = append(buf, counterBytes...)

	// Add some randomness
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	buf = append(buf, randomBytes...)

	// Hash with SHA-256
	hash := sha256.Sum256(buf)

	// Use just the first 16 bytes for shorter ID
	encoded := base64.URLEncoding.EncodeToString(hash[:16])

	// Remove padding and return
	return encoded[:22] // Without padding
}

// GeneratePrefixedID generates an ID with a readable prefix
func GeneratePrefixedID(prefix string) string {
	return prefix + "-" + GenerateShortID()
}

// GenerateUserID creates a user ID (compatible with existing format)
func GenerateUserID() string {
	return GeneratePrefixedID("user")
}

// GenerateLinkID creates a link ID (replacement for the complex existing one)
func GenerateLinkID() string {
	return GenerateID()
}
