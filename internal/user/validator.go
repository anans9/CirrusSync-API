package user

import (
	"cirrussync-api/internal/models"
	"regexp"
	"strings"
)

// NewUserValidator creates a new user validator
func NewUserValidator() UserValidator {
	return &userValidator{}
}

// ValidateCreate validates user creation parameters
func (v *userValidator) ValidateCreate(email, username string, key *UserKey) error {
	println(key.Fingerprint, key.Passphrase, key.Passphrase, key.PassphraseSignature)
	// Validate email
	if !v.ValidateEmail(email) {
		return ErrInvalidEmail
	}

	// Validate username
	if !v.ValidateUsername(username) {
		return ErrInvalidUsername
	}

	// Validate key
	if key == nil {
		return ErrInvalidInput
	}

	if key.PublicKey == "" || key.Passphrase == "" || key.PrivateKey == "" || key.PassphraseSignature == "" || key.Fingerprint == "" {
		return ErrInvalidKey
	}

	return nil
}

// ValidateUpdate validates user update parameters
func (v *userValidator) ValidateUpdate(user *models.User) error {
	if user == nil || user.ID == "" {
		return ErrInvalidInput
	}

	if user.Email != "" && !v.ValidateEmail(user.Email) {
		return ErrInvalidEmail
	}

	if user.Username != "" && !v.ValidateUsername(user.Username) {
		return ErrInvalidUsername
	}

	return nil
}

// ValidateEmail validates an email address
func (v *userValidator) ValidateEmail(email string) bool {
	// Email cannot be empty
	if email == "" {
		return false
	}

	// Normalize before validation
	email = strings.ToLower(strings.TrimSpace(email))

	// Basic email regex pattern
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)

	return match
}

// ValidateUsername validates a username
func (v *userValidator) ValidateUsername(username string) bool {
	// Username cannot be empty
	if username == "" {
		return false
	}

	// Trim spaces before validation
	username = strings.TrimSpace(username)

	// Username must be at least 3 characters and at most 30 characters
	if len(username) < 3 || len(username) > 30 {
		return false
	}

	// Username can only contain alphanumeric characters, underscores, and dashes
	pattern := `^[a-zA-Z0-9_\-]+$`
	match, _ := regexp.MatchString(pattern, username)

	return match
}

// ValidatePlan validates a plan for user assignment
func (v *userValidator) ValidatePlan(plan *models.Plan) error {
	if plan == nil || plan.ID == "" {
		return ErrInvalidInput
	}

	if !plan.Available {
		return ErrPlanNotFound
	}

	return nil
}

// ValidatePaymentMethod validates a payment method
func (v *userValidator) ValidatePaymentMethod(pm *models.UserPaymentMethod) error {
	if pm == nil || pm.UserID == "" {
		return ErrInvalidInput
	}

	if pm.Type == "" {
		return ErrInvalidInput
	}

	// For credit cards
	if pm.Type == "card" {
		if pm.Last4 == "" || pm.Brand == "" || pm.ExpMonth == 0 || pm.ExpYear == 0 {
			return ErrInvalidInput
		}
	}

	return nil
}

// ValidateStorageQuota validates if a storage quota is valid
func (v *userValidator) ValidateStorageQuota(quota int64) error {
	if quota < 0 {
		return ErrInvalidInput
	}

	// Maximum allowed storage (e.g., 10TB)
	maxAllowedStorage := int64(10 * 1024 * 1024 * 1024 * 1024)
	if quota > maxAllowedStorage {
		return ErrInvalidInput
	}

	return nil
}
