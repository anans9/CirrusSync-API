package session

import (
	"errors"
)

var (
	// ErrInvalidInput indicates the provided input is invalid
	ErrInvalidInput = errors.New("Invalid input provided")

	// ErrSessionNotFound indicates the session was not found
	ErrSessionNotFound = errors.New("Session not found")

	// ErrSessionExpired indicates the session has expired
	ErrSessionExpired = errors.New("Session has expired")

	// ErrSessionInvalid indicates the session is marked as invalid
	ErrSessionInvalid = errors.New("Session is invalid")

	// ErrCacheError indicates an error occurred with the Redis cache
	ErrCacheError = errors.New("Cache operation failed")

	// ErrDatabaseError indicates an error occurred with the database
	ErrDatabaseError = errors.New("Database operation failed")

	// ErrUnauthorized indicates the user is not authorized to access the session
	ErrUnauthorized = errors.New("Unauthorized access to session")
)
