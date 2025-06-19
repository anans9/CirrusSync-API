package mfa

import (
	"errors"
	"fmt"
	"time"
)

// Common MFA service errors
var (
	// General errors
	ErrInvalidInput    = errors.New("Invalid input")
	ErrOperationFailed = errors.New("Operation failed")

	// Email verification errors
	ErrInvalidEmail      = errors.New("Invalid email address")
	ErrInvalidToken      = errors.New("Invalid verification token")
	ErrExpiredToken      = errors.New("Verification token has expired")
	ErrEmailExists       = errors.New("A account with this email already exists")
	ErrUsernameExists    = errors.New("Username is taken. Please try another one.")
	ErrEmailAlreadySent  = errors.New("Verification email already sent to this address")
	ErrFailedToSendEmail = errors.New("Failed to send verification email")
	ErrEmailNotVerified  = errors.New("Email address is not verified")
	ErrRateLimited       = errors.New("CirrusSync detected abuse, you are being rate limited. Please visit https://cirrussync.me/abuse for more information.")

	// TOTP errors
	ErrTOTPAlreadyEnabled      = errors.New("TOTP is already enabled for this user")
	ErrTOTPNotEnabled          = errors.New("TOTP is not enabled for this user")
	ErrTOTPNotInitialized      = errors.New("TOTP is not initialized for this user")
	ErrInvalidTOTPCode         = errors.New("Invalid TOTP code")
	ErrTOTPSetupInProgress     = errors.New("TOTP setup is already in progress")
	ErrTOTPOperationInProgress = errors.New("TOTP operation is already in progress")

	// Redis errors
	ErrRateLimitExceeded = errors.New("CirrusSync detected abuse, you are being rate limited. Please visit https://cirrussync.me/abuse for more information.")
)

// TokenGenerationError represents an error that occurred during token generation
type TokenGenerationError struct {
	Err error
}

func (e *TokenGenerationError) Error() string {
	return fmt.Sprintf("failed to generate verification token: %v", e.Err)
}

func (e *TokenGenerationError) Unwrap() error {
	return e.Err
}

// EmailSendError represents an error that occurred during email sending
type EmailSendError struct {
	Email string
	Err   error
}

func (e *EmailSendError) Error() string {
	return fmt.Sprintf("failed to send email to %s: %v", e.Email, e.Err)
}

func (e *EmailSendError) Unwrap() error {
	return e.Err
}

// RateLimitError represents a rate limit error with timing information
type RateLimitError struct {
	Message    string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return e.Message
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}
