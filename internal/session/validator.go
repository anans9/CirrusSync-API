package session

import (
	"cirrussync-api/internal/models"
	"time"
)

// NewSessionValidator creates a new session validator
func NewSessionValidator() SessionValidator {
	return &sessionValidator{}
}

// ValidateSessionCreate validates session creation parameters
func (v *sessionValidator) ValidateSessionCreate(user *models.User, deviceInfo DeviceInfo) error {
	// Validate user
	if user == nil || user.ID == "" {
		return ErrInvalidInput
	}

	// Validate device info
	if deviceInfo.ClientUID == "" {
		return ErrInvalidInput
	}

	return nil
}

// ValidateSession validates an existing session
func (v *sessionValidator) ValidateSession(session *models.UserSession) error {
	// Check if session is nil
	if session == nil {
		return ErrSessionNotFound
	}

	// Check if session is marked as invalid
	if !session.IsValid {
		return ErrSessionInvalid
	}

	// Check if session is expired
	if session.ExpiresAt < time.Now().Unix() {
		return ErrSessionExpired
	}

	return nil
}
