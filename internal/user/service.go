package user

import (
	"cirrussync-api/internal/drive"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/redis"
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

// NewService creates a new user service
func NewService(repo Repository, redisClient *redis.Client, driveService *drive.Service) *Service {
	return &Service{
		repo:         repo,
		driveService: driveService,
		redisClient:  redisClient,
	}
}

// GetUserById retrieves a user by ID with cache lookup
func (s *Service) GetUserById(ctx context.Context, userID string) (*models.User, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	// Try to get from cache first
	user, err := s.getUserFromCache(ctx, userID)
	if err == nil {
		return user, nil
	}

	// Not in cache, get from database
	user, err = s.repo.FindUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Cache the user
	_ = s.cacheUser(ctx, user)

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, ErrInvalidInput
	}

	// Normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	// Try to get user ID from cache
	userID, err := s.getUserIdFromEmailCache(ctx, email)
	if err == nil {
		// Found user ID in cache, get full user
		return s.GetUserById(ctx, userID)
	}

	// Not in cache, get from database
	user, err := s.repo.FindUserOneWhere(&email, nil)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Cache the user
	_ = s.cacheUser(ctx, user)

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *Service) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if username == "" {
		return nil, ErrInvalidInput
	}

	// Try to get user ID from cache
	userID, err := s.getUserIdFromUsernameCache(ctx, username)
	if err == nil {
		// Found user ID in cache, get full user
		return s.GetUserById(ctx, userID)
	}

	// Not in cache, get from database
	user, err := s.repo.FindUserOneWhere(nil, &username)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Cache the user
	_ = s.cacheUser(ctx, user)

	return user, nil
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, email, username string, key UserKey) (*models.User, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Normalize email and username
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	// Validate input
	validator := NewUserValidator()
	if err := validator.ValidateCreate(email, username, &key); err != nil {
		return nil, err
	}

	// Check if email or username exists (perform both checks in parallel)
	emailCh := make(chan error, 1)
	usernameCh := make(chan error, 1)

	go func() {
		existingUserByEmail, err := s.GetUserByEmail(ctx, email)
		if err == nil && existingUserByEmail != nil {
			emailCh <- ErrEmailAlreadyExists
			return
		}
		emailCh <- nil
	}()

	go func() {
		existingUserByUsername, err := s.GetUserByUsername(ctx, username)
		if err == nil && existingUserByUsername != nil {
			usernameCh <- ErrUsernameAlreadyExists
			return
		}
		usernameCh <- nil
	}()

	// Wait for both checks to complete
	if err := <-emailCh; err != nil {
		return nil, err
	}
	if err := <-usernameCh; err != nil {
		return nil, err
	}

	// Create new user
	user := &models.User{
		ID:               utils.GenerateUserID(),
		Email:            email,
		Username:         username,
		DisplayName:      username,
		EmailVerified:    true,
		StripeCustomerID: "jj", // TODO: Generate proper Stripe customer ID
	}

	// Save user first
	savedUser, err := s.repo.SaveUser(user)
	if err != nil {
		s.logger.Error("Failed to save user", "error", err)
		return nil, ErrDatabaseError
	}

	// Create user key - this is critical
	userKey := &models.UserKey{
		ID:                  utils.GenerateLinkID(),
		UserID:              savedUser.ID,
		PublicKey:           key.PublicKey,
		PrivateKey:          key.PrivateKey,
		Passphrase:          key.Passphrase,
		PassphraseSignature: key.PassphraseSignature,
		Fingerprint:         key.Fingerprint,
		Version:             key.Version,
	}

	// Save user key - critical operation
	if err := s.repo.SaveUserKey(userKey); err != nil {
		// If saving user key fails, delete the user and return the error
		deleteErr := s.DeleteUser(ctx, savedUser.ID)
		if deleteErr != nil {
			// Log the delete error, but return the original error that caused the failure
			s.logger.Error("Failed to delete user after key save error",
				"userID", savedUser.ID,
				"deleteError", deleteErr,
				"originalError", err)
		}
		return nil, ErrKeyCreationFailed
	}

	// Create non-critical resources in parallel
	var wg sync.WaitGroup

	// Create default storage (non-critical)
	wg.Add(1)
	go func() {
		defer wg.Done()
		storage := &models.UserStorage{
			ID:     utils.GenerateLinkID(),
			UserID: savedUser.ID,
		}
		if err := s.repo.CreateUserStorage(storage); err != nil {
			s.logger.Warn("Failed to create user storage", "error", err, "userID", savedUser.ID)
			// Non-critical error, continue
		}
	}()

	// Create default preferences and notifications (non-critical)
	wg.Add(1)
	go func() {
		defer wg.Done()
		preferences := &models.UserPreferences{
			ID:     utils.GenerateLinkID(),
			UserID: savedUser.ID,
		}

		if err := s.repo.SaveUserPreferences(preferences); err != nil {
			s.logger.Warn("Failed to create user preferences", "error", err, "userID", savedUser.ID)
			return // Skip notifications if preferences failed
		}

		notifications := &models.UserNotifications{
			ID:            utils.GenerateLinkID(),
			PreferencesID: preferences.ID,
		}

		if err := s.repo.SaveUserNotificationsPreferences(notifications); err != nil {
			s.logger.Warn("Failed to create user notifications", "error", err, "userID", savedUser.ID)
		}
	}()

	// Create default security settings (non-critical)
	wg.Add(1)
	go func() {
		defer wg.Done()
		securitySettings := &models.UserSecuritySettings{
			ID:     utils.GenerateLinkID(),
			UserID: savedUser.ID,
		}

		if err := s.repo.SaveUserSecuritySettings(securitySettings); err != nil {
			s.logger.Warn("Failed to create user security settings", "error", err, "userID", savedUser.ID)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Cache the user (non-critical)
	if err := s.cacheUser(ctx, savedUser); err != nil {
		s.logger.Warn("Failed to cache user", "error", err, "userID", savedUser.ID)
		// Non-critical error, continue
	}

	return savedUser, nil
}

// UpdateUser updates a user
func (s *Service) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) (*models.User, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	// Get current user
	user, err := s.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if displayName, ok := updates["display_name"].(string); ok && displayName != "" {
		user.DisplayName = displayName
	}

	if email, ok := updates["email"].(string); ok && email != "" {
		// Validate email
		validator := NewUserValidator()
		if !validator.ValidateEmail(email) {
			return nil, ErrInvalidEmail
		}

		// Check if email exists for another user
		existingUser, err := s.GetUserByEmail(ctx, email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			return nil, ErrEmailAlreadyExists
		}

		user.Email = email
	}

	if phoneNumber, ok := updates["phone_number"].(string); ok {
		user.PhoneNumber = &phoneNumber
	}

	if companyName, ok := updates["company_name"].(string); ok {
		user.CompanyName = &companyName
	}

	// Update modified time
	user.ModifiedAt = time.Now().Unix()

	// Save to database
	updatedUser, err := s.repo.UpdateUserById(userID, user)
	if err != nil {
		s.logger.Error("Failed to update user", "error", err)
		return nil, ErrDatabaseError
	}

	// Invalidate cache
	_ = s.invalidateUserCache(ctx, userID, user.Email, user.Username)

	// Re-cache updated user
	_ = s.cacheUser(ctx, updatedUser)

	return updatedUser, nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrInvalidInput
	}

	// Get user first (to get email and username for cache invalidation)
	user, err := s.GetUserById(ctx, userID)
	if err != nil {
		return err
	}

	// Delete from database
	err = s.repo.DeleteUser(userID)
	if err != nil {
		s.logger.Error("Failed to delete user", "error", err)
		return ErrDatabaseError
	}

	// Invalidate cache
	_ = s.invalidateUserCache(ctx, userID, user.Email, user.Username)

	return nil
}

// GetUserKeys retrieves a user's keys
func (s *Service) GetUserKeys(ctx context.Context, userID string) ([]*models.UserKey, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	// Try to get from cache first
	keys, err := s.getUserKeysFromCache(ctx, userID)
	if err == nil {
		return keys, nil
	}

	// Not in cache, get from database
	keys, err = s.repo.FindUserKeysByUserID(userID)
	if err != nil {
		s.logger.Error("Failed to get user keys", "error", err)
		return nil, ErrDatabaseError
	}

	// Cache the keys
	_ = s.cacheUserKeys(ctx, userID, keys)

	return keys, nil
}

// AddUserKey adds a new key for a user and deactivates existing keys
func (s *Service) AddUserKey(ctx context.Context, userID string, key UserKey) (*models.UserKey, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	// First, set all existing keys to inactive
	existingKeys, err := s.GetUserKeys(ctx, userID)
	if err != nil && err != ErrUserNotFound {
		s.logger.Error("Failed to get existing keys", "error", err)
		return nil, err
	}

	// Update existing keys to inactive in database
	for _, existingKey := range existingKeys {
		existingKey.Active = false
		existingKey.Primary = false
		existingKey.ModifiedAt = time.Now().Unix()

		err := s.repo.UpdateUserKey(existingKey)
		if err != nil {
			s.logger.Error("Failed to update existing key", "keyID", existingKey.ID, "error", err)
			// Continue with other keys rather than failing completely
		}
	}

	// Create user key
	userKey := &models.UserKey{
		ID:                  utils.GenerateLinkID(),
		UserID:              userID,
		PublicKey:           key.PublicKey,
		PrivateKey:          key.PrivateKey,
		Passphrase:          key.Passphrase,
		PassphraseSignature: key.PassphraseSignature,
		Fingerprint:         key.Fingerprint,
		Version:             key.Version,
		Primary:             true,
		Active:              true,
	}

	// Save user key
	err = s.repo.SaveUserKey(userKey)
	if err != nil {
		s.logger.Error("Failed to save user key", "error", err)
		return nil, ErrDatabaseError
	}

	// Invalidate keys cache
	keysKey := redisKeyForUserKeys(userID)
	_, _ = s.redisClient.Delete(ctx, keysKey)

	return userKey, nil
}

// GetUser retrieves a complete user profile with calculated fields
func (s *Service) GetUser(ctx context.Context, userID string) (User, error) {
	if userID == "" {
		return User{}, ErrInvalidInput
	}

	// Get base user data
	modelUser, err := s.GetUserById(ctx, userID)
	if err != nil {
		return User{}, err
	}

	// Load enriched model with related data
	enrichedModel, err := s.enrichUserData(ctx, modelUser)
	if err != nil {
		return User{}, err
	}

	// Get plans and billing data
	userPlans, err := s.repo.GetUserPlans(userID)
	if err != nil {
		s.logger.Warn("Failed to get user plans", "error", err)
		// Continue without plans rather than failing
		userPlans = []models.UserPlan{}
	}

	// Get latest billing information
	latestBilling, err := s.repo.GetUserBilling(userID)
	if err != nil {
		s.logger.Warn("Failed to get user billing", "error", err)
		// Continue without billing rather than failing
	}

	// Initialize response user
	responseUser := User{
		ID:          enrichedModel.ID,
		Username:    enrichedModel.Username,
		DisplayName: enrichedModel.DisplayName,
		Email:       enrichedModel.Email,
		CreatedAt:   enrichedModel.CreatedAt,
		Currency:    "USD",
	}

	if enrichedModel.Roles != nil {
		if slices.Contains(enrichedModel.Roles, "superadmin") {
			responseUser.Role = 2
		} else if slices.Contains(enrichedModel.Roles, "admin") {
			responseUser.Role = 1
		} else {
			responseUser.Role = 0
		}
	}

	// Handle nullable fields with safe conversion
	if enrichedModel.PhoneNumber != nil {
		responseUser.PhoneNumber = enrichedModel.PhoneNumber
	}

	if enrichedModel.CompanyName != nil {
		responseUser.CompanyName = enrichedModel.CompanyName
	}

	// Set verified status
	responseUser.EmailVerified = enrichedModel.EmailVerified
	responseUser.PhoneVerified = enrichedModel.PhoneVerified

	// Set StripeUser
	responseUser.StripeUser = enrichedModel.StripeUser
	responseUser.StripeUserExists = enrichedModel.StripeUserExists

	// Calculate subscription status based on plans
	responseUser.Subscribed = isActiveSubscription(userPlans)

	// Calculate billing status
	responseUser.Billed = isActiveBilling(latestBilling)

	// Calculate delinquent status
	responseUser.Deliquent = isDelinquentAccount(userPlans)

	// Calculate credit
	var totalCredit float64 = 0
	if enrichedModel.Credits != nil {
		for _, credit := range enrichedModel.Credits {
			if credit.Status == "active" && credit.Active && (credit.ExpiresAt == 0 || credit.ExpiresAt > time.Now().Unix()) {
				totalCredit += float64(credit.Amount)
			}
		}
	}
	responseUser.Credit = totalCredit

	// Set account type based on plan
	responseUser.Type = 1 // Default to free
	for _, plan := range userPlans {
		if plan.Status == "active" {
			switch plan.PlanType {
			case "plus":
				responseUser.Type = 2
			case "pro":
				responseUser.Type = 3
			case "max":
				responseUser.Type = 4
			case "family":
				responseUser.Type = 5
			case "business":
			case "enterprise":
				responseUser.Type = 6
			}
			break
		}
	}

	// Set storage information
	if enrichedModel.Storage != nil {
		responseUser.MaxDriveSpace = int(enrichedModel.Storage.MaxSpace)
		responseUser.UsedDriveSpace = int(enrichedModel.Storage.UsedSpace)
	} else {
		responseUser.MaxDriveSpace = 3221225472 // 3GB default
		responseUser.UsedDriveSpace = 0
	}

	// Process keys
	keyValues := make([]models.UserKey, 0)
	keyPointers, _ := s.GetUserKeys(ctx, userID)
	if len(keyPointers) > 0 {
		// Convert slice of pointers to slice of values
		for _, key := range keyPointers {
			if key != nil {
				keyValues = append(keyValues, *key)
			}
		}
	}

	// Convert to response keys
	responseUser.Keys = convertModelKeysToResponseKeys(keyValues)

	// Set retention flag
	responseUser.RetentionFlag = calculateRetentionFlag(enrichedModel, userPlans)

	// Set onboarding flags
	responseUser.OnboardingFlags = calculateOnboardingFlags(ctx, enrichedModel, s.driveService)

	return responseUser, nil
}

// enrichUserData loads all related data for a user
func (s *Service) enrichUserData(ctx context.Context, user *models.User) (*models.User, error) {
	// Load user's keys
	keyPointers, err := s.GetUserKeys(ctx, user.ID)
	if err != nil {
		s.logger.Error("Failed to get user keys", "error", err)
		// Continue without keys rather than failing completely
	} else if len(keyPointers) > 0 {
		// Convert slice of pointers to slice of values since the model field is []models.UserKey
		keyValues := make([]models.UserKey, len(keyPointers))
		for i, key := range keyPointers {
			if key != nil {
				keyValues[i] = *key
			}
		}
		user.Keys = keyValues
	}
	// Load user's storage information
	storage, err := s.repo.GetUserStorage(user.ID)
	if err == nil && storage != nil {
		user.Storage = storage
	}

	// Load user's preferences
	preferences, err := s.repo.GetUserPreferences(user.ID)
	if err == nil && preferences != nil {
		user.Preferences = preferences
	}

	// Load user's security settings
	securitySettings, err := s.repo.GetUserSecuritySettings(user.ID)
	if err == nil && securitySettings != nil {
		user.SecuritySettings = securitySettings
	}

	// Load user's MFA settings
	mfaSettings, err := s.repo.GetUserMFASettings(user.ID)
	if err == nil && mfaSettings != nil {
		user.MFASettings = mfaSettings
	}

	// Load user's recovery kit
	recoveryKit, err := s.repo.GetUserRecoveryKit(user.ID)
	if err == nil && recoveryKit != nil {
		user.RecoveryKit = recoveryKit
	}

	// Calculate additional fields if needed (for example, credit from credits table)
	credits, err := s.repo.GetUserCredits(user.ID)
	if err != nil {
		s.logger.Warn("Failed to get user credits", "error", err)
		// Continue without credits
	} else if len(credits) > 0 {
		user.Credits = credits
	}

	return user, nil
}

// ValidateEmail is a helper function to validate email format
func (s *Service) ValidateEmail(email string) bool {
	validator := NewUserValidator()
	return validator.ValidateEmail(email)
}

// ValidateUsername is a helper function to validate username format
func (s *Service) ValidateUsername(username string) bool {
	validator := NewUserValidator()
	return validator.ValidateUsername(username)
}
