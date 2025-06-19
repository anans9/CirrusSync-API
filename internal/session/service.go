package session

import (
	"context"
	"time"

	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/redis"
)

// NewService creates a new session service
func NewService(repo Repository, redisClient *redis.Client, logger *logger.Logger) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
		logger:      logger,
	}
}

// IsSessionValid checks if a session is valid
func (s *Service) IsSessionValid(ctx context.Context, sessionID string) bool {
	if sessionID == "" {
		return false
	}

	// Try to get from cache first
	session, err := s.getSessionFromCache(ctx, sessionID)
	if err == nil {
		// Found in cache, validate
		return session.IsValid && session.ExpiresAt > time.Now().Unix()
	}

	// Not in cache, try from database
	session, err = s.repo.GetSession(sessionID)
	if err != nil {
		return false
	}

	// Cache the session for future lookups
	if session.IsValid {
		_ = s.cacheSession(ctx, session)
	}

	// Check if session is expired or invalid
	return session.IsValid && session.ExpiresAt > time.Now().Unix()
}

// InvalidateSession invalidates a session
func (s *Service) InvalidateSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return ErrInvalidInput
	}

	// Try to get from cache first
	cachedSession, err := s.getSessionFromCache(ctx, sessionID)
	userID := ""
	if err == nil {
		userID = cachedSession.UserID
	}

	// Always invalidate cache regardless of database result
	_ = s.invalidateSessionCache(ctx, sessionID, userID)

	// Get from database
	session, err := s.repo.GetSession(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	// Mark as invalid
	session.IsValid = false
	session.ModifiedAt = time.Now().Unix()

	// Save updated session to database
	return s.repo.UpdateSession(session)
}

// CreateSession creates a new session
func (s *Service) CreateSession(ctx context.Context, user *models.User, deviceInfo DeviceInfo, ipAddress string) (*models.UserSession, error) {
	// Validate inputs
	validator := NewSessionValidator()
	if err := validator.ValidateSessionCreate(user, deviceInfo); err != nil {
		return nil, err
	}

	// Create new session
	session := &models.UserSession{
		ID:         utils.GenerateLinkID(),
		UserID:     user.ID,
		Email:      user.Email,
		DeviceName: deviceInfo.ClientName,
		DeviceID:   deviceInfo.ClientUID,
		AppVersion: deviceInfo.AppVersion,
		IPAddress:  ipAddress,
		UserAgent:  deviceInfo.UserAgent,
		ExpiresAt:  time.Now().Add(90 * 24 * time.Hour).Unix(), // 90 days (approximately 3 months)
		CreatedAt:  time.Now().Unix(),
		ModifiedAt: time.Now().Unix(),
		IsValid:    true,
	}

	// Save session to database
	err := s.repo.SaveSession(session)
	if err != nil {
		s.logger.Error("Failed to save session to database", "error", err)
		return nil, ErrDatabaseError
	}

	// Cache the session
	err = s.cacheSession(ctx, session)
	if err != nil {
		s.logger.Warn("Failed to cache session", "error", err)
		// Not returning error as session is already saved to database
	}

	return session, nil
}

// GetSessionByID retrieves a session by ID with cache lookup
func (s *Service) GetSessionByID(ctx context.Context, sessionID string) (*models.UserSession, error) {
	if sessionID == "" {
		return nil, ErrInvalidInput
	}

	// Try to get from cache first
	session, err := s.getSessionFromCache(ctx, sessionID)
	if err == nil {
		// Validate the session
		validator := NewSessionValidator()
		if err := validator.ValidateSession(session); err != nil {
			// If session is invalid or expired, invalidate it
			if err == ErrSessionInvalid || err == ErrSessionExpired {
				_ = s.InvalidateSession(ctx, sessionID)
			}
			return nil, err
		}
		return session, nil
	}

	// Not in cache, get from database
	session, err = s.repo.GetSession(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// Validate the session
	validator := NewSessionValidator()
	if err := validator.ValidateSession(session); err != nil {
		// If session is invalid or expired, invalidate it
		if err == ErrSessionInvalid || err == ErrSessionExpired {
			_ = s.InvalidateSession(ctx, sessionID)
		}
		return nil, err
	}

	// Cache the valid session
	_ = s.cacheSession(ctx, session)

	return session, nil
}

func (s *Service) UpdateSessionByID(ctx context.Context, sessionID string) error {
	session, err := s.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	// Validate the updated session
	validator := NewSessionValidator()
	if err := validator.ValidateSession(session); err != nil {
		return err
	}

	session.LastActive = time.Now().Unix()

	// Update the session in the database
	if err := s.repo.UpdateSession(session); err != nil {
		return ErrDatabaseError
	}

	// Cache the updated session
	_ = s.cacheSession(ctx, session)

	return nil
}

// GetUserSessions retrieves all sessions for a user
func (s *Service) GetUserSessions(ctx context.Context, userID string) ([]*models.UserSession, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	// Get from database
	sessions, err := s.repo.GetAllSessionsByUserID(userID)
	if err != nil {
		return nil, ErrDatabaseError
	}

	// Filter out invalid or expired sessions
	validSessions := make([]*models.UserSession, 0)
	now := time.Now().Unix()

	for _, session := range sessions {
		if session.IsValid && session.ExpiresAt > now {
			validSessions = append(validSessions, session)

			// Cache each valid session
			_ = s.cacheSession(ctx, session)
		}
	}

	return validSessions, nil
}

// RefreshSession extends the expiration time of a session
func (s *Service) RefreshSession(ctx context.Context, sessionID string) error {
	session, err := s.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	// Update expiration time
	session.ExpiresAt = time.Now().Add(90 * 24 * time.Hour).Unix() // 90 days (approximately 3 months)
	session.ModifiedAt = time.Now().Unix()

	// Save to database
	err = s.repo.SaveSession(session)
	if err != nil {
		return ErrDatabaseError
	}

	// Update cache
	_ = s.cacheSession(ctx, session)

	return nil
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (s *Service) InvalidateAllUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrInvalidInput
	}

	// Get all sessions for the user
	sessions, err := s.repo.GetAllSessionsByUserID(userID)
	if err != nil {
		return ErrDatabaseError
	}

	// Invalidate cache for all user sessions
	_ = s.invalidateUserSessionsCache(ctx, userID)

	// Update each session in the database
	now := time.Now().Unix()
	for _, session := range sessions {
		session.IsValid = false
		session.ModifiedAt = now
		err := s.repo.SaveSession(session)
		if err != nil {
			s.logger.Error("Failed to invalidate user session", "sessionID", session.ID, "error", err)
			// Continue with other sessions
		}
	}

	return nil
}

// InvalidateSessionsByDevice invalidates all sessions for a specific device
func (s *Service) InvalidateSessionsByDevice(ctx context.Context, userID string, deviceID string) error {
	if userID == "" || deviceID == "" {
		return ErrInvalidInput
	}

	// Get all sessions for the user
	sessions, err := s.repo.GetAllSessionsByUserID(userID)
	if err != nil {
		return ErrDatabaseError
	}

	// Update each matching session
	now := time.Now().Unix()
	for _, session := range sessions {
		if session.DeviceID == deviceID {
			// Invalidate in cache
			_ = s.invalidateSessionCache(ctx, session.ID, userID)

			// Update in database
			session.IsValid = false
			session.ModifiedAt = now
			err := s.repo.SaveSession(session)
			if err != nil {
				s.logger.Error("Failed to invalidate device session", "sessionID", session.ID, "error", err)
				// Continue with other sessions
			}
		}
	}

	return nil
}
