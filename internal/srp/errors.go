// internal/srp/errors.go
package srp

import (
	"errors"
)

// Error definitions for SRP
var (
	ErrInvalidInput        = errors.New("Invalid input")
	ErrUserAlreadyExists   = errors.New("A account with this email already exists")
	ErrUserNotFound        = errors.New("User not found")
	ErrInvalidCredentials  = errors.New("Invalid credentials")
	ErrRateLimited         = errors.New("CirrusSync detected abuse, you are being rate limited. Please visit https://cirrussync.me/abuse for more information.")
	ErrInvalidSession      = errors.New("Invalid or expired session")
	ErrInvalidServerProof  = errors.New("Invalid server proof")
	ErrInvalidClientProof  = errors.New("Invalid client proof")
	ErrInvalidServerPublic = errors.New("Invalid server public key")
	ErrInvalidClientPublic = errors.New("invalid client public key")
	ErrServerError         = errors.New("Internal server error, please try again later.")
	ErrAuthFailed          = errors.New("Authentication failed")
)
