// internal/jwt/service.go
package jwt

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"cirrussync-api/internal/models"

	"github.com/golang-jwt/jwt/v4"
)

// NewJWTService creates a new JWT service with Ed25519 keys
func NewJWTService(privateKeyPath, publicKeyPath, issuer string, accessExpiry, refreshExpiry time.Duration) (*JWTService, error) {
	// Load private key from cache or file
	privateKey, err := getOrLoadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}
	// Load public key from cache or file
	publicKey, err := getOrLoadPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}
	return &JWTService{
		privateKey:    privateKey,
		publicKey:     publicKey,
		issuer:        issuer,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}, nil
}

// GenerateToken creates a new JWT token with specified expiry and scopes
func (s *JWTService) GenerateToken(userID, email, username string, roles []string, scopes []string, sessionID string, expiry time.Duration, tokenType string, isRefreshToken *bool) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Username:  username,
		Roles:     roles,
		Scopes:    scopes,
		SessionID: sessionID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID,
			ID:        sessionID,
		},
		IsRefreshToken: isRefreshToken,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}

// GenerateTokenPair creates both access and refresh tokens with specified scopes
func (s *JWTService) GenerateTokenPair(userID, email, username string, roles []string, scopes []string, sessionID string) (TokenPair, error) {
	isRefreshToken := true

	// Generate access token
	accessToken, err := s.GenerateToken(userID, email, username, roles, scopes, sessionID, s.accessExpiry, "Bearer", nil)
	if err != nil {
		return TokenPair{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.GenerateToken(userID, email, username, roles, scopes, sessionID, s.refreshExpiry, "Bearer", &isRefreshToken)
	if err != nil {
		return TokenPair{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Scopes:       scopes,
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		// Validate the signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// RefreshTokenPair creates a new token pair using a valid refresh token
func (s *JWTService) RefreshTokenPair(refreshToken string) (TokenPair, error) {
	// Validate refresh token
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return TokenPair{}, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify this is a Bearer token
	if claims.TokenType != "Bearer" {
		return TokenPair{}, errors.New("token is not a refresh token")
	}

	// Generate new token pair, maintaining the same scopes
	return s.GenerateTokenPair(
		claims.UserID,
		claims.Email,
		claims.Username,
		claims.Roles,
		claims.Scopes,
		claims.SessionID,
	)
}

// HasScope checks if a claims object contains a specific scope
func (c *Claims) HasScope(scope string) bool {
	return slices.Contains(c.Scopes, scope)
}

// HasScopes checks if a claims object contains all specified scopes
func (c *Claims) HasScopes(scopes []string) bool {
	for _, requiredScope := range scopes {
		if !c.HasScope(requiredScope) {
			return false
		}
	}
	return true
}

// GetUserScopes returns a standard set of user-related scopes
func GetUserScopes() []string {
	return []string{ScopeUserRead, ScopeUserWrite, ScopeUserExecute, ScopeUserShare}
}

// GetDriveScopes returns a standard set of drive-related scopes
func GetDriveScopes() []string {
	return []string{ScopeDriveRead, ScopeDriveWrite, ScopeDriveShare}
}

// GetAllScopes returns all available scopes
func GetAllScopes() []string {
	return append(GetUserScopes(), GetDriveScopes()...)
}

// GenerateAuthTokens is a convenience function that creates token pairs for authentication
func (s *JWTService) GenerateAuthTokens(user models.User, sessionID string) (TokenPair, error) {
	// Determine scopes based on user roles or other criteria
	var scopes []string

	// Default to all scopes for simplicity, but you could customize based on user.Roles
	scopes = GetUserScopes()

	// For admin users, you might want to ensure they have all scopes
	if contains(user.Roles, "admin") {
		scopes = GetAllScopes()
	}

	return s.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Username,
		user.Roles,
		scopes,
		sessionID,
	)
}

// Helper function to check if a slice contains a value
func contains(slice []string, value string) bool {
	return slices.Contains(slice, value)
}
