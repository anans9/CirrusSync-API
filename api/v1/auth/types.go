package auth

import (
	"cirrussync-api/internal/auth"
	"cirrussync-api/internal/jwt"
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/session"
	"cirrussync-api/internal/user"
)

// Handler manages auth-related HTTP requests
type Handler struct {
	authService    *auth.Service
	userService    *user.Service
	jwtService     *jwt.JWTService
	sessionService *session.Service
	logger         *logger.Logger
}
