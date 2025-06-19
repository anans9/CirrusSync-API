package models

import (
	"time"

	"gorm.io/gorm"

	"cirrussync-api/internal/utils"
)

// UserSRP represents SRP authentication information for a user
type UserSRP struct {
	ID         string `gorm:"primaryKey;column:id"`
	UserID     string `gorm:"column:user_id;not null;unique"`
	Email      string `gorm:"column:email;size:100;not null"`
	Salt       string `gorm:"column:salt;size:255;not null"`
	Verifier   string `gorm:"column:verifier;type:text;not null"`
	Version    int    `gorm:"column:version;default:1"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`
	Active     bool   `gorm:"column:active;default:true;not null"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserSRP
func (UserSRP) TableName() string {
	return "users_srp"
}

// BeforeCreate hook for UserSRP
func (us *UserSRP) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if us.ID == "" {
		us.ID = utils.GenerateLinkID()
	}
	if us.CreatedAt == 0 {
		us.CreatedAt = now
	}
	if us.ModifiedAt == 0 {
		us.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserSRP
func (us *UserSRP) BeforeUpdate(tx *gorm.DB) error {
	us.ModifiedAt = time.Now().Unix()
	return nil
}
