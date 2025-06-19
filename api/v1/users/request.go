package user

// UpdateProfileRequest represents a request to update a user's profile
type UpdateProfileRequest struct {
	DisplayName string `json:"displayName"`
	PhoneNumber string `json:"phoneNumber"`
	CompanyName string `json:"companyName"`
}

// AddKeyRequest represents a request to add a new key
type AddKeyRequest struct {
	PublicKey           string `json:"publicKey" binding:"required"`
	PrivateKey          string `json:"privateKey" binding:"required"`
	Passphrase          string `json:"passphrase" binding:"required"`
	PassphraseSignature string `json:"passphraseSignature" binding:"required"`
	Fingerprint         string `json:"fingerprint" binding:"required"`
	Version             int64  `json:"version" binding:"required"`
}

// RegisterRequest represents a request to register a new user
type RegisterRequest struct {
	Email                string  `json:"email" binding:"required"`
	Username             string  `json:"username" binding:"required"`
	Key                  UserKey `json:"key" binding:"required"`
	SRPSalt              string  `json:"srpSalt" binding:"required"`
	SRPVerifier          string  `json:"srpVerifier" binding:"required"`
	AcceptTermsCondition bool    `json:"acceptTermsCondition" binding:"required"`
}

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	OldSRPSalt     string `json:"oldSrpSalt" binding:"required"`
	OldSRPVerifier string `json:"oldSrpVerifier" binding:"required"`
	NewSRPSalt     string `json:"newSrpSalt" binding:"required"`
	NewSRPVerifier string `json:"newSrpVerifier" binding:"required"`
}

// VerifyEmailRequest represents a request to verify email
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyPhoneRequest represents a request to verify phone
type VerifyPhoneRequest struct {
	Code string `json:"code" binding:"required"`
}

// UpdateEmailRequest represents a request to update email
type UpdateEmailRequest struct {
	Email       string `json:"email" binding:"required"`
	SRPVerifier string `json:"srpVerifier" binding:"required"`
}

// UpdatePreferencesRequest represents a request to update preferences
type UpdatePreferencesRequest struct {
	ThemeMode string `json:"themeMode"`
	Language  string `json:"language"`
	Timezone  string `json:"timezone"`
}

// UpdateSecuritySettingsRequest represents a request to update security settings
type UpdateSecuritySettingsRequest struct {
	TwoFactorRequired           bool `json:"twoFactorRequired"`
	DarkWebMonitoring           bool `json:"darkWebMonitoring"`
	SuspiciousActivityDetection bool `json:"suspiciousActivityDetection"`
	DetailedEvents              bool `json:"detailedEvents"`
}
