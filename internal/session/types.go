package session

import (
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/db"
	"cirrussync-api/pkg/redis"
)

// Service defines the session service interface
type Service struct {
	repo        Repository
	redisClient *redis.Client
	logger      *logger.Logger
}

// Repository defines the session repository interface
type Repository interface {
	// Session operations
	SaveSession(session *models.UserSession) error
	GetSession(sessionID string) (*models.UserSession, error)
	GetAllSessionsByUserID(userID string) ([]*models.UserSession, error)
	UpdateSession(session *models.UserSession) error
	DeleteSession(sessionID string) error

	// User operations
	FindUserByID(id string) (*models.User, error)
	FindUserOneWhere(email *string, username *string) (*models.User, error)
	UpdateUser(user *models.User) (*models.User, error)
}

// DeviceInfo represents device information for session creation
type DeviceInfo struct {
	ClientName string
	ClientUID  string
	AppVersion string
	UserAgent  string
}

// repo is the concrete implementation of Repository
type repo struct {
	userRepo    db.Repository[models.User]
	sessionRepo db.Repository[models.UserSession]
}

// SessionValidator is the interface for session validation
type SessionValidator interface {
	ValidateSessionCreate(user *models.User, deviceInfo DeviceInfo) error
	ValidateSession(session *models.UserSession) error
}

// sessionValidator is the concrete implementation of SessionValidator
type sessionValidator struct{}
