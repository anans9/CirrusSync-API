// internal/jwt/types.go
package jwt

import (
	"crypto/ed25519"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TokenPair represents both access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scopes       []string
	ExpiresIn    int64
}

// Define scope constants to avoid typos and ensure consistency
const (
	ScopeUserRead    = "user-read"
	ScopeUserWrite   = "user-write"
	ScopeUserExecute = "user-execute"
	ScopeUserShare   = "user-share"
	ScopeDriveRead   = "drive-read"
	ScopeDriveWrite  = "drive-write"
	ScopeDriveShare  = "drive-share"
)

// Claims represents the JWT claims for your application
type Claims struct {
	UserID         string
	Email          string
	Username       string
	Roles          []string
	Scopes         []string
	SessionID      string
	TokenType      string
	IsRefreshToken *bool
	jwt.RegisteredClaims
}

// JWTService provides JWT token generation and validation
type JWTService struct {
	privateKey    ed25519.PrivateKey
	publicKey     ed25519.PublicKey
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// Global key cache with lock to ensure thread safety
var (
	keyCache     = make(map[string]any)
	keyCacheLock sync.RWMutex
)
