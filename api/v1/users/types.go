package user

// API Response Types
// BaseResponse provides the base structure for all API responses
type BaseResponse struct {
	Code   int16  `json:"code"`
	Detail string `json:"detail"`
}

// RetentionFlag represents user retention eligibility information
type RetentionFlag struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}

// DriveSetup represents drive setup completion status
type DriveSetup struct {
	Completed bool `json:"completed"`
}

// RootFolder represents root folder creation status
type RootFolder struct {
	Completed bool `json:"completed"`
}

// PlanSelection represents plan selection status
type PlanSelection struct {
	Completed bool `json:"completed"`
}

// OnboardingFlags represents all onboarding status flags
type OnboardingFlags struct {
	DriveSetup    DriveSetup    `json:"driveSetup"`
	RootFolder    RootFolder    `json:"rootFolder"`
	PlanSelection PlanSelection `json:"planSelection"`
}

// Key represents a user encryption key
type Key struct {
	ID                  string `json:"id"`
	Version             int64  `json:"version"`
	Primary             bool   `json:"primary"`
	PrivateKey          string `json:"privateKey"`
	Passphrase          string `json:"passphrase"`
	PassphraseSignature string `json:"passphraseSignature"`
	Fingerprint         string `json:"fingerprint"`
	Active              bool   `json:"active"`
}

// User represents the user data structure for API responses
type User struct {
	ID               string          `json:"id"`
	Username         string          `json:"username"`
	DisplayName      string          `json:"displayName"`
	Email            string          `json:"email"`
	PhoneNumber      string          `json:"phoneNumber"`
	CompanyName      string          `json:"companyName"`
	EmailVerified    bool            `json:"emailVerified"`
	PhoneVerified    bool            `json:"phoneVerified"`
	Currency         string          `json:"currency"`
	Credit           float64         `json:"credit"`
	Type             int             `json:"type"`
	CreatedAt        int64           `json:"createdAt"`
	MaxDriveSpace    int             `json:"maxDriveSpace"`
	UsedDriveSpace   int             `json:"usedDriveSpace"`
	Subscribed       bool            `json:"subscribed"`
	Billed           bool            `json:"billed"`
	Role             string          `json:"role"`
	Deliquent        bool            `json:"deliquent"`
	Keys             []Key           `json:"keys"`
	StripeUser       string          `json:"stripeUser"`
	StripeUserExists bool            `json:"stripeUserExists"`
	RetentionFlag    RetentionFlag   `json:"retentionFlag"`
	OnboardingFlags  OnboardingFlags `json:"onboardingFlags"`
}

// UserKey represents a user's encryption key for internal service operations
type UserKey struct {
	PublicKey           string `json:"publicKey"`
	PrivateKey          string `json:"privateKey"`
	Passphrase          string `json:"passphrase"`
	PassphraseSignature string `json:"passphraseSignature"`
	Fingerprint         string `json:"fingerprint"`
	Version             int    `json:"version"`
}
