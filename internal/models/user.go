package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"cirrussync-api/internal/utils"
)

// User represents a user in the system
type User struct {
	ID               string         `gorm:"primaryKey;column:id"`
	Username         string         `gorm:"column:username;size:50;not null;unique;index:idx_users_username"`
	DisplayName      string         `gorm:"column:display_name;size:50"`
	Email            string         `gorm:"column:email;size:100;not null;unique;index:idx_users_email"`
	EmailVerified    bool           `gorm:"column:email_verified;default:false;not null"`
	PhoneNumber      *string        `gorm:"column:phone_number;size:50;unique;default:null"`
	PhoneVerified    bool           `gorm:"column:phone_verified;default:false"`
	CompanyName      *string        `gorm:"column:company_name;size:100;default:null"`
	CreatedAt        int64          `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt       int64          `gorm:"column:modified_at;autoCreateTime:false;not null"`
	LastLogin        int64          `gorm:"column:last_login;autoCreateTime:false"`
	Roles            []string       `gorm:"column:roles;type:jsonb;serializer:json;default:'[\"user\"]'"`
	Active           bool           `gorm:"column:active;default:true"`
	Deleted          bool           `gorm:"column:deleted;default:false"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at,omitempty"`
	StripeUser       int            `gorm:"column:stripe_user;default:1"`
	StripeUserExists bool           `gorm:"column:stripe_user_exists;default:true"`
	StripeCustomerID string         `gorm:"column:stripe_customer_id;size:100;unique;index:idx_users_stripe_customer_id"`

	// Relationships
	SRP                    []UserSRP               `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Credits                []UserCredit            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	SecurityEvents         []UserSecurityEvent     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	SecuritySettings       *UserSecuritySettings   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Keys                   []UserKey               `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	RecoveryKit            *UserRecoveryKit        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Sessions               []UserSession           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Storage                *UserStorage            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Devices                []UserDevice            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	MFASettings            *UserMFASettings        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Preferences            *UserPreferences        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	BillingInfo            *BillingInfo            `gorm:"foreignKey:UserID"`
	Plans                  []UserPlan              `gorm:"foreignKey:UserID"`
	PaymentMethods         []UserPaymentMethod     `gorm:"foreignKey:UserID"`
	DriveVolumes           []DriveVolume           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	VolumeAllocations      []VolumeAllocation      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	DriveShares            []DriveShare            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	SecurityEventDownloads []SecurityEventDownload `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook for User
func (u *User) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if u.ID == "" {
		u.ID = utils.GenerateUserID()
	}
	if u.CreatedAt == 0 {
		u.CreatedAt = now
	}
	if u.ModifiedAt == 0 {
		u.ModifiedAt = now
	}
	if u.LastLogin == 0 {
		u.LastLogin = now
	}
	return nil
}

// BeforeUpdate hook for User
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.ModifiedAt = time.Now().Unix()
	return nil
}

// UserCredit represents a credit for a user
type UserCredit struct {
	ID                 string          `gorm:"primaryKey;column:id"`
	UserID             string          `gorm:"column:user_id;not null;index:idx_user_credits_user_id_status_active,priority:1"`
	Amount             int             `gorm:"column:amount;default:0"`
	CreatedAt          int64           `gorm:"column:created_at;autoCreateTime:false;not null;index:idx_user_credits_created_at"`
	ModifiedAt         int64           `gorm:"column:modified_at;autoCreateTime:false;not null"`
	ExpiresAt          int64           `gorm:"column:expires_at;index:idx_user_credits_user_id_status_active,priority:5"`
	Status             string          `gorm:"column:status;size:20;default:'active';index:idx_user_credits_user_id_status_active,priority:2"`
	Description        string          `gorm:"column:description;size:255"`
	TransactionID      string          `gorm:"column:transaction_id;index:idx_user_credits_transaction_id"`
	Type               string          `gorm:"column:type;size:20;default:'credit';index:idx_user_credits_user_id_status_active,priority:3"`
	Source             string          `gorm:"column:source;size:50"`
	AdditionalMetadata json.RawMessage `gorm:"column:additional_metadata;type:jsonb;default:'{}'"`
	Active             bool            `gorm:"column:active;default:true;index:idx_user_credits_user_id_status_active,priority:4"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserCredit
func (UserCredit) TableName() string {
	return "users_credits"
}

// BeforeCreate hook for UserCredit
func (uc *UserCredit) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if uc.ID == "" {
		uc.ID = utils.GenerateLinkID()
	}
	if uc.CreatedAt == 0 {
		uc.CreatedAt = now
	}
	if uc.ModifiedAt == 0 {
		uc.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserCredit
func (uc *UserCredit) BeforeUpdate(tx *gorm.DB) error {
	uc.ModifiedAt = time.Now().Unix()
	return nil
}

// UserSecurityEvent represents a security event for a user
type UserSecurityEvent struct {
	ID                 string          `gorm:"primaryKey;column:id"`
	UserID             string          `gorm:"column:user_id;not null;index:idx_security_events_user_id"`
	EventType          string          `gorm:"column:event_type;size:50;not null"`
	Success            bool            `gorm:"column:success;default:true"`
	AdditionalMetadata json.RawMessage `gorm:"column:additional_metadata;type:jsonb;default:'{}'"`
	CreatedAt          int64           `gorm:"column:created_at;autoCreateTime:false;not null;index:idx_security_events_created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserSecurityEvent
func (UserSecurityEvent) TableName() string {
	return "users_security_events"
}

// BeforeCreate hook for UserSecurityEvent
func (use *UserSecurityEvent) BeforeCreate(tx *gorm.DB) error {
	if use.CreatedAt == 0 {
		use.CreatedAt = time.Now().Unix()
	}
	return nil
}

// SecurityEventDownload represents a downloaded security event report
type SecurityEventDownload struct {
	ID           string          `gorm:"primaryKey;column:id"`
	UserID       string          `gorm:"column:user_id;not null;index:idx_downloads_user_id"`
	Status       string          `gorm:"column:status;size:20;default:'processing';not null;index:idx_downloads_status"`
	FileName     string          `gorm:"column:file_name;size:255;not null"`
	FileSize     string          `gorm:"column:file_size;size:50"`
	DownloadURL  string          `gorm:"column:download_url;size:512"`
	ErrorMessage string          `gorm:"column:error_message;type:text"`
	CreatedAt    int64           `gorm:"column:created_at;autoCreateTime:false;not null;index:idx_downloads_created_at"`
	CompletedAt  int64           `gorm:"column:completed_at"`
	ExpiresAt    int64           `gorm:"column:expires_at"`
	Filters      json.RawMessage `gorm:"column:filters;type:jsonb;default:'{}'"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for SecurityEventDownload
func (SecurityEventDownload) TableName() string {
	return "security_event_downloads"
}

// BeforeCreate hook for SecurityEventDownload
func (sed *SecurityEventDownload) BeforeCreate(tx *gorm.DB) error {
	if sed.CreatedAt == 0 {
		sed.CreatedAt = time.Now().Unix()
	}
	return nil
}

// UserSecuritySettings represents security settings for a user
type UserSecuritySettings struct {
	ID                          string `gorm:"primaryKey;column:id"`
	UserID                      string `gorm:"column:user_id;not null;unique;index:idx_security_settings_user_id"`
	DarkWebMonitoring           bool   `gorm:"column:dark_web_monitoring;default:false"`
	DetailedEvents              bool   `gorm:"column:detailed_events;default:true"`
	SuspiciousActivityDetection bool   `gorm:"column:suspicious_activity_detection;default:false"`
	TwoFactorRequired           bool   `gorm:"column:two_factor_required;default:true;index:idx_security_settings_two_factor"`
	CreatedAt                   int64  `gorm:"column:created_at;autoCreateTime:false;not null;index:idx_security_settings_created_at"`
	ModifiedAt                  int64  `gorm:"column:modified_at;autoCreateTime:false;not null;index:idx_security_settings_modified_at"`

	// Relationships
	User           User         `gorm:"foreignKey:UserID"`
	TrustedDevices []UserDevice `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for UserSecuritySettings
func (UserSecuritySettings) TableName() string {
	return "users_security_settings"
}

// BeforeCreate hook for UserSecuritySettings
func (uss *UserSecuritySettings) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if uss.ID == "" {
		uss.ID = utils.GenerateLinkID()
	}
	if uss.CreatedAt == 0 {
		uss.CreatedAt = now
	}
	if uss.ModifiedAt == 0 {
		uss.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserSecuritySettings
func (uss *UserSecuritySettings) BeforeUpdate(tx *gorm.DB) error {
	uss.ModifiedAt = time.Now().Unix()
	return nil
}

// UserKey represents a cryptographic key for a user
type UserKey struct {
	ID                  string `gorm:"primaryKey;column:id"`
	UserID              string `gorm:"column:user_id;not null;index:idx_users_keys_user_id"`
	PrivateKey          string `gorm:"column:private_key;type:text;not null"`
	PublicKey           string `gorm:"column:public_key;type:text;not null"`
	Passphrase          string `gorm:"column:passphrase;type:text;not null"`
	PassphraseSignature string `gorm:"column:passphrase_signature;type:text;not null"`
	Fingerprint         string `gorm:"column:fingerprint;type:text;not null"`
	CreatedAt           int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt          int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`
	RevokedAt           *int64 `gorm:"column:revoked_at;default:null"`
	Primary             bool   `gorm:"column:primary;default:true;index:idx_users_keys_primary"`
	Version             int    `gorm:"column:version;default:1"`
	Active              bool   `gorm:"column:active;default:true;index:idx_users_keys_active"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserKey
func (UserKey) TableName() string {
	return "users_keys"
}

// BeforeCreate hook for UserKey
func (uk *UserKey) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if uk.ID == "" {
		uk.ID = utils.GenerateLinkID()
	}
	if uk.CreatedAt == 0 {
		uk.CreatedAt = now
	}
	if uk.ModifiedAt == 0 {
		uk.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserKey
func (uk *UserKey) BeforeUpdate(tx *gorm.DB) error {
	uk.ModifiedAt = time.Now().Unix()
	return nil
}

// UserRecoveryKit represents a recovery kit for a user
type UserRecoveryKit struct {
	ID              string          `gorm:"primaryKey;column:id"`
	UserID          string          `gorm:"column:user_id;not null;unique;index:idx_recovery_kits_user_id"`
	AccountRecovery json.RawMessage `gorm:"column:account_recovery;type:jsonb;not null"`
	DataRecovery    json.RawMessage `gorm:"column:data_recovery;type:jsonb;not null"`
	CreatedAt       int64           `gorm:"column:created_at;autoCreateTime:false;not null"`
	LastUpdated     int64           `gorm:"column:last_updated"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserRecoveryKit
func (UserRecoveryKit) TableName() string {
	return "users_recovery_kits"
}

// BeforeCreate hook for UserRecoveryKit
func (urk *UserRecoveryKit) BeforeCreate(tx *gorm.DB) error {
	if urk.ID == "" {
		urk.ID = utils.GenerateLinkID()
	}
	if urk.CreatedAt == 0 {
		urk.CreatedAt = time.Now().Unix()
	}
	return nil
}

// UserSession represents a session for a user
type UserSession struct {
	ID         string `gorm:"primaryKey;column:id"`
	UserID     string `gorm:"column:user_id;not null;index:idx_sessions_user_expires,priority:1"`
	Email      string `gorm:"column:email;not null"`
	Username   string `gorm:"column:username;not null"`
	DeviceID   string `gorm:"column:device_id;not null"`
	DeviceName string `gorm:"column:device_name;size:100"`
	AppVersion string `gorm:"column:app_version;size:20"`
	IPAddress  string `gorm:"column:ip_address;size:45"`
	UserAgent  string `gorm:"column:user_agent;size:255"`
	ExpiresAt  int64  `gorm:"column:expires_at;not null;index:idx_sessions_user_expires,priority:2;index:idx_sessions_inactive,priority:2"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`
	IsValid    bool   `gorm:"column:is_valid;default:true;not null"`
	LastActive int64  `gorm:"column:last_active;autoCreateTime:false;not null;index:idx_sessions_inactive,priority:1"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserSession
func (UserSession) TableName() string {
	return "users_sessions"
}

// BeforeCreate hook for UserSession
func (us *UserSession) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if us.ID == "" {
		us.ID = utils.GenerateLinkID()
	}
	if us.CreatedAt == 0 {
		us.CreatedAt = now
	}
	if us.LastActive == 0 {
		us.LastActive = now
	}
	return nil
}

// BeforeUpdate hook for UserCredit
func (us *UserSession) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().Unix()
	us.ModifiedAt = now
	us.LastActive = now
	return nil
}

// UserStorage represents storage information for a user
type UserStorage struct {
	ID            string `gorm:"primaryKey;column:id"`
	UserID        string `gorm:"column:user_id;not null;unique;index:idx_user_storage_user_id"`
	UsedSpace     int64  `gorm:"column:used_space;default:0"`
	MaxSpace      int64  `gorm:"column:max_space;default:3221225472"`       // 3GB default
	BasePlanSpace int64  `gorm:"column:base_plan_space;default:3221225472"` // Base space from plan
	SharedSpace   int64  `gorm:"column:shared_space;default:0"`             // Space from shared volumes
	CreatedAt     int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt    int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserStorage
func (UserStorage) TableName() string {
	return "user_storage"
}

// BeforeCreate hook for UserStorage
func (us *UserStorage) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if us.ID == "" {
		us.ID = utils.GenerateLinkID()
	}
	if us.CreatedAt == 0 {
		us.CreatedAt = now
	}
	if us.ModifiedAt == 0 {
		us.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserStorage
func (us *UserStorage) BeforeUpdate(tx *gorm.DB) error {
	us.ModifiedAt = time.Now().Unix()
	return nil
}

// UserDevice represents a device for a user
type UserDevice struct {
	ID           string          `gorm:"primaryKey;column:id"`
	UserID       string          `gorm:"column:user_id;not null;index:idx_users_devices_user_id"`
	DeviceID     string          `gorm:"column:device_id;not null;index:idx_users_devices_device_id"`
	DeviceName   string          `gorm:"column:device_name;size:100"`
	DeviceType   string          `gorm:"column:device_type;size:50"`
	Trusted      bool            `gorm:"column:trusted;default:false;index:idx_users_devices_trusted"`
	LastUsed     int64           `gorm:"column:last_used"`
	CreatedAt    int64           `gorm:"column:created_at;autoCreateTime:false;not null"`
	RecoveryData json.RawMessage `gorm:"column:recovery_data;type:jsonb;serializer:json"`
	Active       bool            `gorm:"column:active;default:true"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserDevice
func (UserDevice) TableName() string {
	return "users_devices"
}

// BeforeCreate hook for UserDevice
func (ud *UserDevice) BeforeCreate(tx *gorm.DB) error {
	if ud.ID == "" {
		ud.ID = utils.GenerateLinkID()
	}
	if ud.CreatedAt == 0 {
		ud.CreatedAt = time.Now().Unix()
	}
	return nil
}

// EmailMethods represents email authentication method settings
type EmailMethods struct {
	ID         string `gorm:"primaryKey;column:id"`
	SettingsID string `gorm:"column:settings_id;index:idx_email_methods_settings_id"`
	Enabled    bool   `gorm:"column:enabled;default:false" json:"enabled"`
	Verified   bool   `gorm:"column:verified;default:false" json:"verified"`
	LastUsed   *int64 `gorm:"column:last_used;default:null" json:"last_used"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	Settings UserMFASettings `gorm:"foreignKey:SettingsID"`
}

// TableName specifies the table name for EmailMethods
func (EmailMethods) TableName() string {
	return "email_methods"
}

// BeforeCreate hook for EmailMethods
func (e *EmailMethods) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()

	if e.ID == "" {
		e.ID = utils.GenerateLinkID()
	}
	if e.CreatedAt == 0 {
		e.CreatedAt = now
	}
	if e.ModifiedAt == 0 {
		e.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for EmailMethods
func (e *EmailMethods) BeforeUpdate(tx *gorm.DB) error {
	e.ModifiedAt = time.Now().Unix()
	return nil
}

// PhoneMethods represents phone authentication method settings
type PhoneMethods struct {
	ID         string `gorm:"primaryKey;column:id"`
	SettingsID string `gorm:"column:settings_id;index:idx_phone_methods_settings_id"`
	Enabled    bool   `gorm:"column:enabled;default:false" json:"enabled"`
	Verified   bool   `gorm:"column:verified;default:false" json:"verified"`
	LastUsed   *int64 `gorm:"column:last_used;default:null" json:"last_used"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	Settings UserMFASettings `gorm:"foreignKey:SettingsID"`
}

// TableName specifies the table name for PhoneMethods
func (PhoneMethods) TableName() string {
	return "phone_methods"
}

// BeforeCreate hook for PhoneMethods
func (p *PhoneMethods) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()

	if p.ID == "" {
		p.ID = utils.GenerateLinkID()
	}
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	if p.ModifiedAt == 0 {
		p.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for PhoneMethods
func (p *PhoneMethods) BeforeUpdate(tx *gorm.DB) error {
	p.ModifiedAt = time.Now().Unix()
	return nil
}

// TOTPMethods represents TOTP authentication method settings
type TOTPMethods struct {
	ID         string `gorm:"primaryKey;column:id"`
	SettingsID string `gorm:"column:settings_id;index:idx_totp_methods_settings_id"`
	Secret     string `gorm:"column:secret" json:"-"` // Omitted from JSON responses
	Enabled    bool   `gorm:"column:enabled;default:false" json:"enabled"`
	Verified   bool   `gorm:"column:verified;default:false" json:"verified"`
	LastUsed   *int64 `gorm:"column:last_used;default:null" json:"last_used"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	Settings UserMFASettings `gorm:"foreignKey:SettingsID"`
}

// TableName specifies the table name for TOTPMethods
func (TOTPMethods) TableName() string {
	return "totp_methods"
}

// BeforeCreate hook for TOTPMethods
func (t *TOTPMethods) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()

	if t.ID == "" {
		t.ID = utils.GenerateLinkID()
	}
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	if t.ModifiedAt == 0 {
		t.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for PhoneMethods
func (t *TOTPMethods) BeforeUpdate(tx *gorm.DB) error {
	t.ModifiedAt = time.Now().Unix()
	return nil
}

// UserMFASettings represents MFA settings for a user
type UserMFASettings struct {
	ID              string   `gorm:"primaryKey;column:id"`
	UserID          string   `gorm:"column:user_id;not null;unique;index:idx_users_mfa_settings_user_id"`
	PreferredMethod string   `gorm:"column:preferred_method;size:10;index:idx_users_mfa_settings_preferred_method"`
	BackupCodes     []string `gorm:"column:backup_codes;type:jsonb;serializer:json;default:'[]'" json:"-"` // Omitted from JSON responses
	CreatedAt       int64    `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt      int64    `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	User  User          `gorm:"foreignKey:UserID"`
	Email *EmailMethods `gorm:"foreignKey:SettingsID"`
	Phone *PhoneMethods `gorm:"foreignKey:SettingsID"`
	TOTP  *TOTPMethods  `gorm:"foreignKey:SettingsID"`
}

// TableName specifies the table name for UserMFASettings
func (UserMFASettings) TableName() string {
	return "users_mfa_settings"
}

// BeforeCreate hook for UserMFASettings
func (ums *UserMFASettings) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()

	if ums.ID == "" {
		ums.ID = utils.GenerateLinkID()
	}
	if ums.CreatedAt == 0 {
		ums.CreatedAt = now
	}
	if ums.ModifiedAt == 0 {
		ums.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserMFASettings
func (ums *UserMFASettings) BeforeUpdate(tx *gorm.DB) error {
	ums.ModifiedAt = time.Now().Unix()
	return nil
}

// UserNotifications represents notification preferences for a user
type UserNotifications struct {
	ID            string `gorm:"primaryKey;column:id"`
	PreferencesID string `gorm:"column:preferences_id;not null;unique;index:idx_user_notifications_preferences_id"`
	Email         bool   `gorm:"column:email;default:true" json:"email"`
	Push          bool   `gorm:"column:push;default:true" json:"push"`
	Security      bool   `gorm:"column:security;default:true" json:"security"`
	CreatedAt     int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt    int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	Preferences UserPreferences `gorm:"foreignKey:PreferencesID"`
}

// TableName specifies the table name for UserNotifications
func (UserNotifications) TableName() string {
	return "user_notifications"
}

// BeforeCreate hook for UserNotifications
func (un *UserNotifications) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if un.ID == "" {
		un.ID = utils.GenerateLinkID()
	}
	if un.CreatedAt == 0 {
		un.CreatedAt = now
	}
	if un.ModifiedAt == 0 {
		un.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserNotifications
func (un *UserNotifications) BeforeUpdate(tx *gorm.DB) error {
	un.ModifiedAt = time.Now().Unix()
	return nil
}

// UserPreferences represents preferences for a user
type UserPreferences struct {
	ID         string `gorm:"primaryKey;column:id"`
	UserID     string `gorm:"column:user_id;not null;unique;index:ix_users_preferences_user_id"`
	ThemeMode  string `gorm:"column:theme_mode;size:10;default:'system';index:ix_users_preferences_theme_mode"`
	Language   string `gorm:"column:language;size:10;default:'en';index:ix_users_preferences_language"`
	Timezone   string `gorm:"column:timezone;size:50;default:'UTC'"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	User          User               `gorm:"foreignKey:UserID"`
	Notifications *UserNotifications `gorm:"foreignKey:PreferencesID"`
}

// TableName specifies the table name for UserPreferences
func (UserPreferences) TableName() string {
	return "users_preferences"
}

// BeforeCreate hook for UserPreferences
func (up *UserPreferences) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if up.ID == "" {
		up.ID = utils.GenerateLinkID()
	}
	if up.CreatedAt == 0 {
		up.CreatedAt = now
	}
	if up.ModifiedAt == 0 {
		up.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserPreferences
func (up *UserPreferences) BeforeUpdate(tx *gorm.DB) error {
	up.ModifiedAt = time.Now().Unix()
	return nil
}
