// internal/srp/repository.go
package srp

import (
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/db"
	"context"

	"gorm.io/gorm"
)

// Repository interface defines required methods for SRP data access
type Repository interface {
	SaveUserSRP(userSRP *models.UserSRP) error
	UpdateUserSRP(userSRP *models.UserSRP) error
	GetUserSRP(userID string) (*models.UserSRP, error)
	GetUserSRPByEmail(email string) (*models.UserSRP, error)
}

// repo implements the Repository interface
type repo struct {
	baseRepo *db.BaseRepository[models.UserSRP]
}

// NewRepository creates a new SRP repository
func NewRepository(database *gorm.DB) Repository {
	// Create a base repository with the DB connection
	baseRepo := db.NewRepositoryWithDB[models.UserSRP](database)
	// Return our repository that wraps the base repository
	return &repo{
		baseRepo: baseRepo,
	}
}

// SaveUserSRP creates new SRP credentials
func (r *repo) SaveUserSRP(userSRP *models.UserSRP) error {
	return r.baseRepo.Create(context.Background(), userSRP)
}

// UpdateUserSRP updates existing SRP credentials
func (r *repo) UpdateUserSRP(userSRP *models.UserSRP) error {
	return r.baseRepo.Update(context.Background(), userSRP)
}

// GetUserSRP retrieves SRP credentials by user ID
func (r *repo) GetUserSRP(userID string) (*models.UserSRP, error) {
	var userSRP models.UserSRP
	err := r.baseRepo.DB().Where("user_id = ?", userID).First(&userSRP).Error
	if err != nil {
		return nil, err
	}
	return &userSRP, nil
}

// GetUserSRPByEmail retrieves SRP credentials by email
func (r *repo) GetUserSRPByEmail(email string) (*models.UserSRP, error) {
	var userSRP models.UserSRP
	err := r.baseRepo.DB().Where("email = ?", email).First(&userSRP).Error
	if err != nil {
		return nil, err
	}
	return &userSRP, nil
}
