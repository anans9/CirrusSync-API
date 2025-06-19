package user

import (
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/db"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// NewRepository creates a new user repository
func NewRepository(database *gorm.DB) Repository {
	// Create base repositories with the DB connection
	userRepo := db.NewRepositoryWithDB[models.User](database)
	userKeyRepo := db.NewRepositoryWithDB[models.UserKey](database)
	userStorageRepo := db.NewRepositoryWithDB[models.UserStorage](database)
	userPreferencesRepo := db.NewRepositoryWithDB[models.UserPreferences](database)
	userNotificationsRepo := db.NewRepositoryWithDB[models.UserNotifications](database)
	userSecuritySettingsRepo := db.NewRepositoryWithDB[models.UserSecuritySettings](database)
	userMFASettingsRepo := db.NewRepositoryWithDB[models.UserMFASettings](database)
	userRecoveryKitRepo := db.NewRepositoryWithDB[models.UserRecoveryKit](database)
	userCreditRepo := db.NewRepositoryWithDB[models.UserCredit](database)
	userPlanRepo := db.NewRepositoryWithDB[models.UserPlan](database)
	userBillingRepo := db.NewRepositoryWithDB[models.UserBilling](database)
	userPaymentMethodRepo := db.NewRepositoryWithDB[models.UserPaymentMethod](database)
	userDeviceRepo := db.NewRepositoryWithDB[models.UserDevice](database)

	// Return our repository that wraps the base repositories
	return &repo{
		db:                       database,
		userRepo:                 userRepo,
		userKeyRepo:              userKeyRepo,
		userStorageRepo:          userStorageRepo,
		userPreferencesRepo:      userPreferencesRepo,
		userNotificationsRepo:    userNotificationsRepo,
		userSecuritySettingsRepo: userSecuritySettingsRepo,
		userMFASettingsRepo:      userMFASettingsRepo,
		userRecoveryKitRepo:      userRecoveryKitRepo,
		userCreditRepo:           userCreditRepo,
		userPlanRepo:             userPlanRepo,
		userBillingRepo:          userBillingRepo,
		userPaymentMethodRepo:    userPaymentMethodRepo,
		userDeviceRepo:           userDeviceRepo,
	}
}

// repo is the concrete implementation of Repository
type repo struct {
	db                       *gorm.DB
	userRepo                 db.Repository[models.User]
	userKeyRepo              db.Repository[models.UserKey]
	userStorageRepo          db.Repository[models.UserStorage]
	userPreferencesRepo      db.Repository[models.UserPreferences]
	userNotificationsRepo    db.Repository[models.UserNotifications]
	userSecuritySettingsRepo db.Repository[models.UserSecuritySettings]
	userMFASettingsRepo      db.Repository[models.UserMFASettings]
	userRecoveryKitRepo      db.Repository[models.UserRecoveryKit]
	userCreditRepo           db.Repository[models.UserCredit]
	userPlanRepo             db.Repository[models.UserPlan]
	userBillingRepo          db.Repository[models.UserBilling]
	userPaymentMethodRepo    db.Repository[models.UserPaymentMethod]
	userDeviceRepo           db.Repository[models.UserDevice]
}

// USER OPERATIONS

// SaveUser creates a new user with built-in locking
func (r *repo) SaveUser(user *models.User) (*models.User, error) {
	err := r.userRepo.Create(context.Background(), user)
	return user, err
}

// UpdateUserById updates a user with built-in locking
func (r *repo) UpdateUserById(id string, user *models.User) (*models.User, error) {
	err := r.userRepo.Update(context.Background(), user)
	return user, err
}

// FindUserByID finds a user by ID
func (r *repo) FindUserByID(id string) (*models.User, error) {
	return r.userRepo.FindByID(context.Background(), id)
}

// FindUserOneWhere finds a user by email or username
func (r *repo) FindUserOneWhere(email *string, username *string) (*models.User, error) {
	var user models.User

	// First check by email if provided
	if email != nil {
		err := r.userRepo.DB().Where("email = ?", *email).First(&user).Error
		if err == nil {
			return &user, nil
		}
		// If it's not a "record not found" error, return the error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Then check by username if provided
	if username != nil {
		err := r.userRepo.DB().Where("username = ?", *username).First(&user).Error
		if err == nil {
			return &user, nil
		}
		// Return whatever error occurred, including "record not found"
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// If we get here, no user was found by either email or username
	return nil, gorm.ErrRecordNotFound
}

// DeleteUser deletes a user with built-in locking
func (r *repo) DeleteUser(id string) error {
	return r.userRepo.Delete(context.Background(), id)
}

// KEY OPERATIONS

// SaveUserKey saves a user key
func (r *repo) SaveUserKey(userKey *models.UserKey) error {
	return r.userKeyRepo.Create(context.Background(), userKey)
}

// FindUserKeysByUserID finds all keys for a user
func (r *repo) FindUserKeysByUserID(userID string) ([]*models.UserKey, error) {
	var keys []*models.UserKey
	err := r.userKeyRepo.DB().Where("user_id = ?", userID).Find(&keys).Error
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// UpdateUserKey updates a user key
func (r *repo) UpdateUserKey(userKey *models.UserKey) error {
	return r.userKeyRepo.Update(context.Background(), userKey)
}

// DeleteUserKey deletes a user key
func (r *repo) DeleteUserKey(id string) error {
	return r.userKeyRepo.Delete(context.Background(), id)
}

// STORAGE OPERATIONS

// CreateUserStorage creates storage information for a user
func (r *repo) CreateUserStorage(storage *models.UserStorage) error {
	return r.userStorageRepo.Create(context.Background(), storage)
}

// GetUserStorage gets storage information for a user
func (r *repo) GetUserStorage(userID string) (*models.UserStorage, error) {
	var storage models.UserStorage
	err := r.userStorageRepo.DB().Where("user_id = ?", userID).First(&storage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default storage for the user
			storage = models.UserStorage{
				UserID:        userID,
				UsedSpace:     0,
				MaxSpace:      3221225472, // 3GB default
				BasePlanSpace: 3221225472,
				SharedSpace:   0,
				CreatedAt:     time.Now().Unix(),
				ModifiedAt:    time.Now().Unix(),
			}
			err = r.userStorageRepo.Create(context.Background(), &storage)
			if err != nil {
				return nil, err
			}
			return &storage, nil
		}
		return nil, err
	}
	return &storage, nil
}

// UpdateUserStorage updates storage information for a user
func (r *repo) UpdateUserStorage(storage *models.UserStorage) error {
	return r.userStorageRepo.Update(context.Background(), storage)
}

// PREFERENCES OPERATIONS

// SaveUserPreferences saves preferences for a user
func (r *repo) SaveUserPreferences(preferences *models.UserPreferences) error {
	return r.userPreferencesRepo.Create(context.Background(), preferences)
}

// GetUserPreferences gets preferences for a user
func (r *repo) GetUserPreferences(userID string) (*models.UserPreferences, error) {
	var preferences models.UserPreferences
	err := r.userPreferencesRepo.DB().Where("user_id = ?", userID).First(&preferences).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default preferences for the user
			preferences = models.UserPreferences{
				UserID:     userID,
				ThemeMode:  "system",
				Language:   "en",
				Timezone:   "UTC",
				CreatedAt:  time.Now().Unix(),
				ModifiedAt: time.Now().Unix(),
			}
			err = r.userPreferencesRepo.Create(context.Background(), &preferences)
			if err != nil {
				return nil, err
			}
			return &preferences, nil
		}
		return nil, err
	}
	return &preferences, nil
}

// UpdateUserPreferences updates preferences for a user
func (r *repo) UpdateUserPreferences(preferences *models.UserPreferences) error {
	return r.userPreferencesRepo.Update(context.Background(), preferences)
}

// Preferences Notifications

func (r *repo) SaveUserNotificationsPreferences(notifications *models.UserNotifications) error {
	return r.userNotificationsRepo.Create(context.Background(), notifications)
}

func (r *repo) UpdateUserNotificationsPreferences(notifications *models.UserNotifications) error {
	return r.userNotificationsRepo.Update(context.Background(), notifications)
}

func (r *repo) GetUserNotificationsPreferences(userID string) (*models.UserNotifications, error) {
	return r.userNotificationsRepo.FindByID(context.Background(), userID)
}

// Security Settings Operations

// SaveUserSecuritySettings saves security settings for a user
func (r *repo) SaveUserSecuritySettings(settings *models.UserSecuritySettings) error {
	return r.userSecuritySettingsRepo.Create(context.Background(), settings)
}

// GetUserSecuritySettings gets security settings for a user
func (r *repo) GetUserSecuritySettings(userID string) (*models.UserSecuritySettings, error) {
	return r.userSecuritySettingsRepo.FindByID(context.Background(), userID)
}

// UpdateUserSecuritySettings updates security settings for a user
func (r *repo) UpdateUserSecuritySettings(settings *models.UserSecuritySettings) error {
	return r.userSecuritySettingsRepo.Update(context.Background(), settings)
}

// MFA SETTINGS OPERATIONS

// Create GetUserMFASettings
func (r *repo) SaveUserMFASettings(mfaSettings *models.UserMFASettings) error {
	return r.userMFASettingsRepo.Create(context.Background(), mfaSettings)
}

// GetUserMFASettings gets MFA settings for a user
func (r *repo) GetUserMFASettings(userID string) (*models.UserMFASettings, error) {
	var settings models.UserMFASettings
	err := r.userMFASettingsRepo.DB().Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User doesn't have MFA settings yet
			return nil, nil
		}
		return nil, err
	}
	return &settings, nil
}

// UpdateUserMFASettings updates MFA settings for a user
func (r *repo) UpdateUserMFASettings(settings *models.UserMFASettings) error {
	return r.userMFASettingsRepo.Update(context.Background(), settings)
}

// RECOVERY KIT OPERATIONS

// GetUserRecoveryKit gets the recovery kit for a user
func (r *repo) GetUserRecoveryKit(userID string) (*models.UserRecoveryKit, error) {
	var kit models.UserRecoveryKit
	err := r.userRecoveryKitRepo.DB().Where("user_id = ?", userID).First(&kit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User doesn't have a recovery kit yet
			return nil, nil
		}
		return nil, err
	}
	return &kit, nil
}

// UpdateUserRecoveryKit updates the recovery kit for a user
func (r *repo) UpdateUserRecoveryKit(kit *models.UserRecoveryKit) error {
	return r.userRecoveryKitRepo.Update(context.Background(), kit)
}

// CREDITS OPERATIONS

// GetUserCredits gets all credits for a user
func (r *repo) GetUserCredits(userID string) ([]models.UserCredit, error) {
	var credits []models.UserCredit
	err := r.userCreditRepo.DB().Where("user_id = ? AND status = ? AND active = ?", userID, "active", true).Find(&credits).Error
	if err != nil {
		return nil, err
	}
	return credits, nil
}

// SaveUserCredit saves a user credit
func (r *repo) SaveUserCredit(credit *models.UserCredit) error {
	return r.userCreditRepo.Create(context.Background(), credit)
}

// BILLING OPERATIONS

// GetUserBilling gets the billing information for a user
func (r *repo) GetUserBilling(userID string) (*models.UserBilling, error) {
	var billing models.UserBilling
	err := r.userBillingRepo.DB().Where("user_id = ?", userID).Order("created_at DESC").First(&billing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User doesn't have billing information yet
			return nil, nil
		}
		return nil, err
	}
	return &billing, nil
}

// UpdateUserBilling updates the billing information for a user
func (r *repo) UpdateUserBilling(billing *models.UserBilling) error {
	return r.userBillingRepo.Update(context.Background(), billing)
}

// PLAN OPERATIONS

// GetUserPlans gets all plans for a user
func (r *repo) GetUserPlans(userID string) ([]models.UserPlan, error) {
	var plans []models.UserPlan
	err := r.userPlanRepo.DB().Where("user_id = ?", userID).Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetActivePlan gets the active plan for a user
func (r *repo) GetActivePlan(userID string) (*models.UserPlan, error) {
	var plan models.UserPlan
	now := time.Now().Unix()
	err := r.userPlanRepo.DB().Where(
		"user_id = ? AND status = ? AND current_period_start <= ? AND (current_period_end = 0 OR current_period_end > ?)",
		userID, "active", now, now,
	).Order("created_at DESC").First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User doesn't have an active plan
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

// SaveUserPlan saves a user plan
func (r *repo) SaveUserPlan(plan *models.UserPlan) error {
	return r.userPlanRepo.Create(context.Background(), plan)
}

// UpdateUserPlan updates a user plan
func (r *repo) UpdateUserPlan(plan *models.UserPlan) error {
	return r.userPlanRepo.Update(context.Background(), plan)
}

// DEVICE OPERATIONS

// GetUserDevices gets all devices for a user
func (r *repo) GetUserDevices(userID string) ([]models.UserDevice, error) {
	var devices []models.UserDevice
	err := r.userDeviceRepo.DB().Where("user_id = ? AND active = ?", userID, true).Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// SaveUserDevice saves a user device
func (r *repo) SaveUserDevice(device *models.UserDevice) error {
	return r.userDeviceRepo.Create(context.Background(), device)
}

// UpdateUserDevice updates a user device
func (r *repo) UpdateUserDevice(device *models.UserDevice) error {
	return r.userDeviceRepo.Update(context.Background(), device)
}

// DeleteUserDevice deletes a user device
func (r *repo) DeleteUserDevice(id string) error {
	return r.userDeviceRepo.Delete(context.Background(), id)
}

// PAYMENT METHOD OPERATIONS

// GetUserPaymentMethods gets all payment methods for a user
func (r *repo) GetUserPaymentMethods(userID string) ([]models.UserPaymentMethod, error) {
	var methods []models.UserPaymentMethod
	err := r.userPaymentMethodRepo.DB().Where("user_id = ? AND active = ?", userID, true).Find(&methods).Error
	if err != nil {
		return nil, err
	}
	return methods, nil
}

// SaveUserPaymentMethod saves a user payment method
func (r *repo) SaveUserPaymentMethod(method *models.UserPaymentMethod) error {
	return r.userPaymentMethodRepo.Create(context.Background(), method)
}

// UpdateUserPaymentMethod updates a user payment method
func (r *repo) UpdateUserPaymentMethod(method *models.UserPaymentMethod) error {
	return r.userPaymentMethodRepo.Update(context.Background(), method)
}

// DeleteUserPaymentMethod deletes a user payment method
func (r *repo) DeleteUserPaymentMethod(id string) error {
	return r.userPaymentMethodRepo.Delete(context.Background(), id)
}
