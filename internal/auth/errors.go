package auth

import (
	"errors"
)

// Custom error types for the auth package
var (
	// ErrInvalidInput indicates the provided input is invalid
	ErrInvalidInput = errors.New("Invalid input provided")

	// ErrInvalidCredentials indicates the credentials are invalid
	ErrInvalidCredentials = errors.New("Invalid credentials")

	// ErrInvalidEmail indicates the provided email is invalid
	ErrInvalidEmail = errors.New("Invalid email format")

	// ErrInvalidUsername indicates the provided username is invalid
	ErrInvalidUsername = errors.New("Invalid username format")

	// ErrEmailAlreadyExists indicates the email is already in use
	ErrEmailAlreadyExists = errors.New("Email already exists")

	// ErrUsernameAlreadyExists indicates the username is already in use
	ErrUsernameAlreadyExists = errors.New("Username already exists")
)
