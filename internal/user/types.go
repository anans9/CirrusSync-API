package user

import (
	"cirrussync-api/internal/drive"
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/redis"
)

// Service defines the user service
type Service struct {
	repo         Repository
	redisClient  *redis.Client
	driveService *drive.Service
	logger       *logger.Logger
}

// Repository defines the user repository interface
type Repository interface {
	// User operations
	SaveUser(user *models.User) (*models.User, error)
	UpdateUserById(id string, user *models.User) (*models.User, error)
	FindUserByID(id string) (*models.User, error)
	FindUserOneWhere(email *string, username *string) (*models.User, error)
	DeleteUser(id string) error

	// Key operations
	SaveUserKey(userKey *models.UserKey) error
	FindUserKeysByUserID(userID string) ([]*models.UserKey, error)
	UpdateUserKey(userKey *models.UserKey) error
	DeleteUserKey(id string) error

	// Storage operations
	CreateUserStorage(storage *models.UserStorage) error
	GetUserStorage(userID string) (*models.UserStorage, error)
	UpdateUserStorage(storage *models.UserStorage) error

	// Preferences operations
	SaveUserPreferences(preferences *models.UserPreferences) error
	GetUserPreferences(userID string) (*models.UserPreferences, error)
	UpdateUserPreferences(preferences *models.UserPreferences) error

	// Notifications operations
	SaveUserNotificationsPreferences(notifications *models.UserNotifications) error
	GetUserNotificationsPreferences(userID string) (*models.UserNotifications, error)
	UpdateUserNotificationsPreferences(notifications *models.UserNotifications) error

	// Security settings operations
	SaveUserSecuritySettings(settings *models.UserSecuritySettings) error
	GetUserSecuritySettings(userID string) (*models.UserSecuritySettings, error)
	UpdateUserSecuritySettings(settings *models.UserSecuritySettings) error

	// MFA settings operations
	SaveUserMFASettings(mfaSettings *models.UserMFASettings) error
	GetUserMFASettings(userID string) (*models.UserMFASettings, error)
	UpdateUserMFASettings(settings *models.UserMFASettings) error

	// Recovery kit operations
	GetUserRecoveryKit(userID string) (*models.UserRecoveryKit, error)
	UpdateUserRecoveryKit(kit *models.UserRecoveryKit) error

	// Credits operations
	GetUserCredits(userID string) ([]models.UserCredit, error)
	SaveUserCredit(credit *models.UserCredit) error

	// Billing operations
	GetUserBilling(userID string) (*models.UserBilling, error)
	UpdateUserBilling(billing *models.UserBilling) error

	// Plan operations
	GetUserPlans(userID string) ([]models.UserPlan, error)
	SaveUserPlan(plan *models.UserPlan) error
	UpdateUserPlan(plan *models.UserPlan) error

	// Device operations
	GetUserDevices(userID string) ([]models.UserDevice, error)
	SaveUserDevice(device *models.UserDevice) error
	UpdateUserDevice(device *models.UserDevice) error
	DeleteUserDevice(id string) error
}

// UserKey represents a user's encryption key
type UserKey struct {
	Version             int    `json:"version"`
	PublicKey           string `json:"publicKey"`
	PrivateKey          string `json:"privateKey"`
	Passphrase          string `json:"passphrase"`
	PassphraseSignature string `json:"passphraseSignature"`
	Fingerprint         string `json:"fingerprint"`
}

// UserValidator is the interface for user validation
type UserValidator interface {
	ValidateCreate(email, username string, key *UserKey) error
	ValidateUpdate(user *models.User) error
	ValidateEmail(email string) bool
	ValidateUsername(username string) bool
}

// userValidator is the concrete implementation of UserValidator
type userValidator struct{}

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
	Version             int    `json:"version"`
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
	PhoneNumber      *string         `json:"phoneNumber"`
	CompanyName      *string         `json:"companyName"`
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
	Role             int             `json:"role"`
	Deliquent        bool            `json:"deliquent"`
	Keys             []Key           `json:"keys"`
	StripeUser       int             `json:"stripeUser"`
	StripeUserExists bool            `json:"stripeUserExists"`
	RetentionFlag    RetentionFlag   `json:"retentionFlag"`
	OnboardingFlags  OnboardingFlags `json:"onboardingFlags"`
}

// UserResponse is the complete response structure for user-related API responses
type UserResponse struct {
	BaseResponse
	User User `json:"user"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	BaseResponse
	Error string `json:"error,omitempty"`
}
