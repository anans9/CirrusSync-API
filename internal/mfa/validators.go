package mfa

import (
	"regexp"
	"strings"
)

var (
	emailRegex       = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	totpCodeRegex    = regexp.MustCompile(`^[0-9]{6}$`)
	recoveryKeyRegex = regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`)
)

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	email = strings.TrimSpace(email)
	return emailRegex.MatchString(email)
}

// ValidateTOTPCode validates a TOTP code format
func ValidateTOTPCode(code string) bool {
	if code == "" {
		return false
	}

	code = strings.TrimSpace(code)

	// Check if it's a 6-digit code
	if totpCodeRegex.MatchString(code) {
		return true
	}

	// Or check if it's a recovery key format (XXXX-XXXX-XXXX-XXXX)
	if recoveryKeyRegex.MatchString(code) {
		return true
	}

	return false
}

// NormalizeEmail normalizes an email address
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizeTOTPCode normalizes a TOTP code
func NormalizeTOTPCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(code, " ", "")))
}

// isValidIntent checks if the intent is valid
func isValidIntent(intent string) bool {
	return intent == "signup" || intent == "password-reset" || intent == "2fa"
}
