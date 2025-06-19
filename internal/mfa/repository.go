// internal/auth/repository.go
package mfa

import (
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/db"

	"errors"

	"gorm.io/gorm"
)

// Repository interface defines required methods
type Repository interface {
	// User
	FindUserOneWhere(email *string, username *string) (*models.User, error)
}

// It uses our base repository to inherit locking capabilities
type repo struct {
	userRepo *db.BaseRepository[models.User]
}

// NewRepository creates a new user repository
func NewRepository(database *gorm.DB) Repository {
	// Create base repositories with the DB connection
	userRepo := db.NewRepositoryWithDB[models.User](database)
	// Return our repository that wraps the base repositories
	return &repo{
		userRepo: userRepo,
	}
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
