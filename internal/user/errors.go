package user

import (
	"errors"
)

// Custom error types for the user package
var (
	// ErrInvalidInput indicates the provided input is invalid
	ErrInvalidInput = errors.New("Invalid input provided")

	// ErrUserNotFound indicates the user was not found
	ErrUserNotFound = errors.New("User not found")

	// ErrInvalidEmail indicates the provided email is invalid
	ErrInvalidEmail = errors.New("Invalid email format")

	// ErrInvalidUsername indicates the provided username is invalid
	ErrInvalidUsername = errors.New("Invalid username format")

	// ErrEmailAlreadyExists indicates the email is already in use
	ErrEmailAlreadyExists = errors.New("Email already exists")

	// ErrUsernameAlreadyExists indicates the username is already in use
	ErrUsernameAlreadyExists = errors.New("Username already exists")

	// ErrCacheError indicates an error occurred with the Redis cache
	ErrCacheError = errors.New("Cache operation failed")

	// ErrDatabaseError indicates an error occurred with the database
	ErrDatabaseError = errors.New("Database operation failed")

	// ErrKeyCreationFailed indicates an error in creating a user key
	ErrKeyCreationFailed = errors.New("User key creation failed")

	// ErrInvalidKey indicates the provided key is invalid
	ErrInvalidKey = errors.New("Provided key is invalid")

	// ErrNoActiveKey indicates the user has no active key
	ErrNoActiveKey = errors.New("No active key found for user")

	// ErrStorageNotFound indicates the user's storage information was not found
	ErrStorageNotFound = errors.New("User storage not found")

	// ErrPreferencesNotFound indicates the user's preferences were not found
	ErrPreferencesNotFound = errors.New("User preferences not found")

	// ErrSecuritySettingsNotFound indicates the user's security settings were not found
	ErrSecuritySettingsNotFound = errors.New("User security settings not found")

	// ErrMFASettingsNotFound indicates the user's MFA settings were not found
	ErrMFASettingsNotFound = errors.New("User MFA settings not found")

	// ErrRecoveryKitNotFound indicates the user's recovery kit was not found
	ErrRecoveryKitNotFound = errors.New("User recovery kit not found")

	// ErrBillingInfoNotFound indicates the user's billing information was not found
	ErrBillingInfoNotFound = errors.New("User billing information not found")

	// ErrPlanNotFound indicates the user's plan information was not found
	ErrPlanNotFound = errors.New("User plan not found")

	// ErrPlanActivationFailed indicates plan activation failed
	ErrPlanActivationFailed = errors.New("Plan activation failed")

	// ErrPlanDeactivationFailed indicates plan deactivation failed
	ErrPlanDeactivationFailed = errors.New("Plan deactivation failed")

	// ErrUnauthorized indicates the user is not authorized for the operation
	ErrUnauthorized = errors.New("User not authorized for this operation")

	// ErrAccountDeactivated indicates the user account is deactivated
	ErrAccountDeactivated = errors.New("User account is deactivated")

	// ErrAccountLocked indicates the user account is locked
	ErrAccountLocked = errors.New("User account is locked")
)
