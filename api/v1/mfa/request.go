package mfa

// SendVerificationRequest represents a request to send an email verification
type SendVerificationRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Intent   string `json:"intent" binding:"required,oneof=signup password-reset 2fa"`
}

// VerifyEmailRequest represents a request to verify an email
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// CheckVerificationRequest represents a request to check if an email is verified
type CheckVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// GenerateTOTPRequest represents a request to generate TOTP
type GenerateTOTPRequest struct {
	UserID string `json:"userId" binding:"required"`
}

// VerifyTOTPRequest represents a request to verify TOTP
type VerifyTOTPRequest struct {
	UserID string `json:"userId" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

// ValidateTOTPRequest represents a request to validate TOTP
type ValidateTOTPRequest struct {
	UserID string `json:"userId" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

// DisableTOTPRequest represents a request to disable TOTP
type DisableTOTPRequest struct {
	UserID           string `json:"userId" binding:"required"`
	ConfirmationCode string `json:"confirmationCode"`
}
