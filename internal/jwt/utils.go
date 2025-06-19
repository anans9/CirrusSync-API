// internal/jwt/utils.go
package jwt

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v4"
)

// getOrLoadPrivateKey loads a private key from cache or from file if not cached
func getOrLoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	// Check cache first using read lock
	keyCacheLock.RLock()
	cachedKey, exists := keyCache["private:"+path]
	keyCacheLock.RUnlock()

	if exists {
		return cachedKey.(ed25519.PrivateKey), nil
	}

	// Key not in cache, load from file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if block.Type != "PRIVATE KEY" {
		return nil, errors.New("not a private key")
	}

	privateKey, err := jwt.ParseEdPrivateKeyFromPEM(data)
	if err != nil {
		return nil, err
	}

	key := privateKey.(ed25519.PrivateKey)

	// Store in cache using write lock
	keyCacheLock.Lock()
	keyCache["private:"+path] = key
	keyCacheLock.Unlock()

	return key, nil
}

// getOrLoadPublicKey loads a public key from cache or from file if not cached
func getOrLoadPublicKey(path string) (ed25519.PublicKey, error) {
	// Check cache first using read lock
	keyCacheLock.RLock()
	cachedKey, exists := keyCache["public:"+path]
	keyCacheLock.RUnlock()

	if exists {
		return cachedKey.(ed25519.PublicKey), nil
	}

	// Key not in cache, load from file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if block.Type != "PUBLIC KEY" {
		return nil, errors.New("not a public key")
	}

	publicKey, err := jwt.ParseEdPublicKeyFromPEM(data)
	if err != nil {
		return nil, err
	}

	key := publicKey.(ed25519.PublicKey)

	// Store in cache using write lock
	keyCacheLock.Lock()
	keyCache["public:"+path] = key
	keyCacheLock.Unlock()

	return key, nil
}

// GenerateKeyPair generates a new Ed25519 key pair and saves to PEM files
func GenerateKeyPair(privateKeyPath, publicKeyPath string) error {
	// Generate Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Encode private key to PKCS8 format and then to PEM
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	privatePEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privateFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open private key file: %w", err)
	}
	defer privateFile.Close()

	if err := pem.Encode(privateFile, privatePEM); err != nil {
		return fmt.Errorf("failed to encode private key to PEM: %w", err)
	}

	// Encode public key to PKIX format and then to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open public key file: %w", err)
	}
	defer publicFile.Close()

	if err := pem.Encode(publicFile, publicPEM); err != nil {
		return fmt.Errorf("failed to encode public key to PEM: %w", err)
	}

	// Update the cache with the new keys
	keyCacheLock.Lock()
	keyCache["private:"+privateKeyPath] = privateKey
	keyCache["public:"+publicKeyPath] = publicKey
	keyCacheLock.Unlock()

	return nil
}
