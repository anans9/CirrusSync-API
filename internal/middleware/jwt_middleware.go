package middleware

import (
	"cirrussync-api/internal/jwt"
	"cirrussync-api/internal/session"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware creates a middleware for JWT authentication
func JWTAuthMiddleware(jwtService *jwt.JWTService, sessionService *session.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if this is the refresh token endpoint
		isRefreshEndpoint := c.FullPath() == "/api/v1/auth/refresh" || strings.HasSuffix(c.Request.URL.Path, "/refresh")

		// Try to extract tokens efficiently - check common sources once
		accessTokenString := extractTokenFromSources(c, "Authorization", "accessToken")
		refreshTokenString := extractTokenFromSources(c, "X-Refresh-Token", "refreshToken")

		// If refresh token not found in header, try normal Authorization for refresh token
		if refreshTokenString == "" && isRefreshEndpoint {
			refreshTokenString = accessTokenString // Try to use the token from Authorization as refresh token
		}

		// Early exit if no tokens found
		if accessTokenString == "" && refreshTokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Authentication required"})
			c.Abort()
			return
		}

		// Try access token first if available
		accessTokenValid := false
		if accessTokenString != "" {
			claims, err := jwtService.ValidateToken(accessTokenString)
			if err == nil && sessionService.IsSessionValid(c.Request.Context(), claims.SessionID) {
				// Valid access token and session, set claims and proceed
				setClaimsInContext(c, claims)
				accessTokenValid = true

				// For non-refresh endpoints, just proceed with valid access token
				if !isRefreshEndpoint {
					c.Next()
					return
				}
				// For refresh endpoint, continue checking refresh token too
			}
		}

		// For non-refresh endpoints, if access token is invalid, don't allow fallback to refresh token
		if !accessTokenValid && !isRefreshEndpoint {
			c.JSON(http.StatusUnauthorized, gin.H{
				"detail": "Access token expired or invalid",
				"code":   "token_expired", // Special code to signal client to refresh
			})
			c.Abort()
			return
		}

		// For refresh endpoint or if we need to validate refresh token
		if refreshTokenString != "" {
			refreshClaims, err := jwtService.ValidateToken(refreshTokenString)
			if err == nil {
				// Safely check if this is a refresh token
				isRefreshToken := false
				if refreshClaims.IsRefreshToken != nil {
					isRefreshToken = *refreshClaims.IsRefreshToken
				}

				sessionID := refreshClaims.SessionID
				if isRefreshToken && sessionID != "" && sessionService.IsSessionValid(c.Request.Context(), sessionID) {
					// Set essential claims from refresh token
					c.Set("userID", refreshClaims.UserID)
					c.Set("email", refreshClaims.Email)
					c.Set("username", refreshClaims.Username)
					c.Set("scopes", refreshClaims.Scopes)
					c.Set("sessionID", sessionID)
					c.Set("claims", refreshClaims.RegisteredClaims)
					c.Set("isRefreshToken", true)

					// Extract roles (if available)
					rolesInterface := refreshClaims.Roles
					if len(rolesInterface) > 0 {
						c.Set("roles", convertInterfaceToStringSlice(rolesInterface))
					}

					// Only allow proceeding with just refresh token for the refresh endpoint
					if isRefreshEndpoint {
						c.Next()
						return
					}
				}
			}
		}

		// If we're here and it's the refresh endpoint, that means refresh token was invalid
		if isRefreshEndpoint {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid refresh token"})
			c.Abort()
			return
		}

		// For all other endpoints, access token was already checked and found invalid
		// This is a fallback that shouldn't normally be reached
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Authentication failed"})
		c.Abort()
	}
}

// Helper function to extract token from multiple sources (header or cookie)
func extractTokenFromSources(c *gin.Context, headerName, cookieName string) string {
	header := c.GetHeader(headerName)
	if headerName == "Authorization" && header != "" {
		parts := strings.Split(header, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	} else if header != "" {
		return header
	}

	cookie, err := c.Cookie(cookieName)
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
}

// Helper function to set claims in context
func setClaimsInContext(c *gin.Context, claims *jwt.Claims) {
	c.Set("userID", claims.UserID)
	c.Set("email", claims.Email)
	c.Set("username", claims.Username)
	c.Set("roles", claims.Roles)
	c.Set("scopes", claims.Scopes)
	c.Set("sessionID", claims.SessionID)
	c.Set("claims", claims.RegisteredClaims)

	// Safely set isRefreshToken
	isRefreshToken := false
	if claims.IsRefreshToken != nil {
		isRefreshToken = *claims.IsRefreshToken
	}
	c.Set("isRefreshToken", isRefreshToken)
}

// Helper function to convert interface{} to []string - O(n) where n is number of roles
func convertInterfaceToStringSlice(value interface{}) []string {
	result := []string{}

	if slice, ok := value.([]interface{}); ok {
		for _, item := range slice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	} else if slice, ok := value.([]string); ok {
		return slice
	}

	return result
}

// RoleRequiredMiddleware creates a middleware that requires specific roles
func RoleRequiredMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the roles from the context (set by JWTAuthMiddleware)
		rolesInterface, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Authentication required"})
			c.Abort()
			return
		}

		roles, ok := rolesInterface.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "Internal server error"})
			c.Abort()
			return
		}

		// Check if the user has any of the required roles
		hasRequiredRole := false
		for _, requiredRole := range requiredRoles {
			if slices.Contains(roles, requiredRole) {
				hasRequiredRole = true
				break
			}
		}

		if !hasRequiredRole {
			c.JSON(http.StatusForbidden, gin.H{"detail": "You don't have permission to access this resource"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminRequiredMiddleware creates a middleware that requires admin role
func AdminRequiredMiddleware() gin.HandlerFunc {
	return RoleRequiredMiddleware("admin")
}

// OptionalJWTAuthMiddleware attempts to validate JWT but continues if not present
func OptionalJWTAuthMiddleware(jwtService *jwt.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromSources(c, "Authorization", "accessToken")

		// If no token found, just continue
		if tokenString == "" {
			c.Next()
			return
		}

		// Validate the token
		claims, err := jwtService.ValidateToken(tokenString)
		if err == nil {
			// Valid token, set claims
			setClaimsInContext(c, claims)
			c.Set("authenticated", true)
		}

		// Continue regardless of token validity
		c.Next()
	}
}
