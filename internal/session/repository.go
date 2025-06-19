package session

import (
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/db"
	"context"
	"errors"

	"gorm.io/gorm"
)

// NewRepository creates a new session repository
func NewRepository(database *gorm.DB) Repository {
	// Create base repositories with the DB connection
	userRepo := db.NewRepositoryWithDB[models.User](database)
	sessionRepo := db.NewRepositoryWithDB[models.UserSession](database)

	// Return our repository that wraps the base repositories
	return &repo{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Update updates a user with built-in locking
func (r *repo) UpdateUser(user *models.User) (*models.User, error) {
	// This uses the Update method from the base repository
	// which already includes FOR UPDATE locking
	err := r.userRepo.Update(context.Background(), user)
	return user, err
}

// FindByID finds a user by ID
func (r *repo) FindUserByID(id string) (*models.User, error) {
	return r.userRepo.FindByID(context.Background(), id)
}

// FindOneWhere finds a user by email or username
func (r *repo) FindUserOneWhere(email *string, username *string) (*models.User, error) {
	var user models.User

	// First check by email if provided
	if email != nil {
		err := r.userRepo.DB().Where("email = ?", *email).First(&user).Error
		if err == nil {
			return &user, nil
		}
		// If it's not a "record not found" error, return the error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Then check by username if provided
	if username != nil {
		err := r.userRepo.DB().Where("username = ?", *username).First(&user).Error
		if err == nil {
			return &user, nil
		}
		// Return whatever error occurred, including "record not found"
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// If we get here, no user was found by either email or username
	return nil, gorm.ErrRecordNotFound
}

// SaveSession creates or updates a session
func (r *repo) SaveSession(session *models.UserSession) error {
	// Check if session exists
	var existingSession models.UserSession
	err := r.sessionRepo.DB().Where(&models.UserSession{ID: session.ID}).First(&existingSession).Error

	if err == nil {
		// Update existing session
		return r.sessionRepo.Update(context.Background(), session)
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new session
		return r.sessionRepo.Create(context.Background(), session)
	} else {
		// Any other error
		return err
	}
}

// GetSession retrieves a session by ID
func (r *repo) GetSession(sessionID string) (*models.UserSession, error) {
	return r.sessionRepo.FindByID(context.Background(), sessionID)
}

// GetAllSessions retrieves all sessions for a user ID
func (r *repo) GetAllSessionsByUserID(userID string) ([]*models.UserSession, error) {
	var sessions []*models.UserSession
	err := r.sessionRepo.DB().Where("user_id = ?", userID).Find(&sessions).Error
	return sessions, err
}

// UpdateSession updates a session
func (r *repo) UpdateSession(session *models.UserSession) error {
	return r.sessionRepo.Update(context.Background(), session)
}

// DeleteSession deletes a session
func (r *repo) DeleteSession(sessionID string) error {
	return r.sessionRepo.Delete(context.Background(), sessionID)
}
