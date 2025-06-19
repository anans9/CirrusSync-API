package user

import (
	"cirrussync-api/internal/drive"
	"cirrussync-api/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Redis key generators
func redisKeyForUser(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func redisKeyForUserByEmail(email string) string {
	return fmt.Sprintf("user:email:%s", email)
}

func redisKeyForUserByUsername(username string) string {
	return fmt.Sprintf("user:username:%s", username)
}

func redisKeyForUserKeys(userID string) string {
	return fmt.Sprintf("user:%s:keys", userID)
}

func redisKeyForUserStorage(userID string) string {
	return fmt.Sprintf("user:%s:storage", userID)
}

func redisKeyForUserPreferences(userID string) string {
	return fmt.Sprintf("user:%s:preferences", userID)
}

func redisKeyForUserSecuritySettings(userID string) string {
	return fmt.Sprintf("user:%s:security", userID)
}

func redisKeyForUserMFASettings(userID string) string {
	return fmt.Sprintf("user:%s:mfa", userID)
}

func redisKeyForUserRecoveryKit(userID string) string {
	return fmt.Sprintf("user:%s:recovery", userID)
}

func redisKeyForUserCredits(userID string) string {
	return fmt.Sprintf("user:%s:credits", userID)
}

func redisKeyForUserPlans(userID string) string {
	return fmt.Sprintf("user:%s:plans", userID)
}

func redisKeyForUserBilling(userID string) string {
	return fmt.Sprintf("user:%s:billing", userID)
}

// Cache operations
func (s *Service) cacheUser(ctx context.Context, user *models.User) error {
	// Marshal user to JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		s.logger.Error("Failed to marshal user for caching", "error", err)
		return ErrCacheError
	}

	// Cache with an hour expiration
	userKey := redisKeyForUser(user.ID)
	err = s.redisClient.Set(ctx, userKey, string(userJSON), time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache user", "error", err)
		return ErrCacheError
	}

	// Cache email and username lookups
	emailKey := redisKeyForUserByEmail(user.Email)
	err = s.redisClient.Set(ctx, emailKey, user.ID, time.Hour)
	if err != nil {
		s.logger.Warn("Failed to cache user email lookup", "error", err)
		// Not returning error as this is not critical
	}

	usernameKey := redisKeyForUserByUsername(user.Username)
	err = s.redisClient.Set(ctx, usernameKey, user.ID, time.Hour)
	if err != nil {
		s.logger.Warn("Failed to cache user username lookup", "error", err)
		// Not returning error as this is not critical
	}

	return nil
}

func (s *Service) getUserFromCache(ctx context.Context, userID string) (*models.User, error) {
	// Get from cache
	userKey := redisKeyForUser(userID)
	userJSON, err := s.redisClient.Get(ctx, userKey)
	if err != nil || userJSON == "" {
		return nil, ErrUserNotFound
	}

	// Unmarshal JSON to user
	var user models.User
	err = json.Unmarshal([]byte(userJSON), &user)
	if err != nil {
		s.logger.Error("Failed to unmarshal cached user", "error", err)
		return nil, ErrCacheError
	}

	return &user, nil
}

func (s *Service) getUserIdFromEmailCache(ctx context.Context, email string) (string, error) {
	emailKey := redisKeyForUserByEmail(email)
	userID, err := s.redisClient.Get(ctx, emailKey)
	if err != nil || userID == "" {
		return "", ErrUserNotFound
	}

	return userID, nil
}

func (s *Service) getUserIdFromUsernameCache(ctx context.Context, username string) (string, error) {
	usernameKey := redisKeyForUserByUsername(username)
	userID, err := s.redisClient.Get(ctx, usernameKey)
	if err != nil || userID == "" {
		return "", ErrUserNotFound
	}

	return userID, nil
}

func (s *Service) cacheUserKeys(ctx context.Context, userID string, keys []*models.UserKey) error {
	// Marshal keys to JSON
	keysJSON, err := json.Marshal(keys)
	if err != nil {
		s.logger.Error("Failed to marshal user keys for caching", "error", err)
		return ErrCacheError
	}

	// Cache with an hour expiration
	keysKey := redisKeyForUserKeys(userID)
	err = s.redisClient.Set(ctx, keysKey, string(keysJSON), time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache user keys", "error", err)
		return ErrCacheError
	}

	return nil
}

func (s *Service) getUserKeysFromCache(ctx context.Context, userID string) ([]*models.UserKey, error) {
	// Get from cache
	keysKey := redisKeyForUserKeys(userID)
	keysJSON, err := s.redisClient.Get(ctx, keysKey)
	if err != nil || keysJSON == "" {
		return nil, ErrUserNotFound
	}

	// Unmarshal JSON to keys
	var keys []*models.UserKey
	err = json.Unmarshal([]byte(keysJSON), &keys)
	if err != nil {
		s.logger.Error("Failed to unmarshal cached user keys", "error", err)
		return nil, ErrCacheError
	}

	return keys, nil
}

func (s *Service) cacheUserPlans(ctx context.Context, userID string, plans []models.UserPlan) error {
	// Marshal plans to JSON
	plansJSON, err := json.Marshal(plans)
	if err != nil {
		s.logger.Error("Failed to marshal user plans for caching", "error", err)
		return ErrCacheError
	}

	// Cache with an hour expiration
	plansKey := redisKeyForUserPlans(userID)
	err = s.redisClient.Set(ctx, plansKey, string(plansJSON), time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache user plans", "error", err)
		return ErrCacheError
	}

	return nil
}

func (s *Service) getUserPlansFromCache(ctx context.Context, userID string) ([]models.UserPlan, error) {
	// Get from cache
	plansKey := redisKeyForUserPlans(userID)
	plansJSON, err := s.redisClient.Get(ctx, plansKey)
	if err != nil || plansJSON == "" {
		return nil, ErrUserNotFound
	}

	// Unmarshal JSON to plans
	var plans []models.UserPlan
	err = json.Unmarshal([]byte(plansJSON), &plans)
	if err != nil {
		s.logger.Error("Failed to unmarshal cached user plans", "error", err)
		return nil, ErrCacheError
	}

	return plans, nil
}

func (s *Service) cacheUserBilling(ctx context.Context, userID string, billing *models.UserBilling) error {
	if billing == nil {
		return nil
	}

	// Marshal billing to JSON
	billingJSON, err := json.Marshal(billing)
	if err != nil {
		s.logger.Error("Failed to marshal user billing for caching", "error", err)
		return ErrCacheError
	}

	// Cache with an hour expiration
	billingKey := redisKeyForUserBilling(userID)
	err = s.redisClient.Set(ctx, billingKey, string(billingJSON), time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache user billing", "error", err)
		return ErrCacheError
	}

	return nil
}

func (s *Service) getUserBillingFromCache(ctx context.Context, userID string) (*models.UserBilling, error) {
	// Get from cache
	billingKey := redisKeyForUserBilling(userID)
	billingJSON, err := s.redisClient.Get(ctx, billingKey)
	if err != nil || billingJSON == "" {
		return nil, ErrUserNotFound
	}

	// Unmarshal JSON to billing
	var billing models.UserBilling
	err = json.Unmarshal([]byte(billingJSON), &billing)
	if err != nil {
		s.logger.Error("Failed to unmarshal cached user billing", "error", err)
		return nil, ErrCacheError
	}

	return &billing, nil
}

func (s *Service) invalidateUserCache(ctx context.Context, userID string, email string, username string) error {
	// Delete user key
	userKey := redisKeyForUser(userID)
	_, err := s.redisClient.Delete(ctx, userKey)
	if err != nil {
		s.logger.Warn("Failed to remove user from cache", "error", err)
	}

	// Delete email and username lookup keys if provided
	if email != "" {
		emailKey := redisKeyForUserByEmail(email)
		_, _ = s.redisClient.Delete(ctx, emailKey)
	}

	if username != "" {
		usernameKey := redisKeyForUserByUsername(username)
		_, _ = s.redisClient.Delete(ctx, usernameKey)
	}

	// Delete all related cache keys
	keysKey := redisKeyForUserKeys(userID)
	_, _ = s.redisClient.Delete(ctx, keysKey)

	storageKey := redisKeyForUserStorage(userID)
	_, _ = s.redisClient.Delete(ctx, storageKey)

	preferencesKey := redisKeyForUserPreferences(userID)
	_, _ = s.redisClient.Delete(ctx, preferencesKey)

	securityKey := redisKeyForUserSecuritySettings(userID)
	_, _ = s.redisClient.Delete(ctx, securityKey)

	mfaKey := redisKeyForUserMFASettings(userID)
	_, _ = s.redisClient.Delete(ctx, mfaKey)

	recoveryKey := redisKeyForUserRecoveryKit(userID)
	_, _ = s.redisClient.Delete(ctx, recoveryKey)

	creditsKey := redisKeyForUserCredits(userID)
	_, _ = s.redisClient.Delete(ctx, creditsKey)

	plansKey := redisKeyForUserPlans(userID)
	_, _ = s.redisClient.Delete(ctx, plansKey)

	billingKey := redisKeyForUserBilling(userID)
	_, _ = s.redisClient.Delete(ctx, billingKey)

	return nil
}

// Helper functions for response model conversion
func convertModelKeysToResponseKeys(modelKeys []models.UserKey) []Key {
	keys := make([]Key, len(modelKeys))
	for i, modelKey := range modelKeys {
		keys[i] = Key{
			ID:                  modelKey.ID,
			Version:             modelKey.Version,
			Primary:             modelKey.Primary,
			PrivateKey:          modelKey.PrivateKey,
			Passphrase:          modelKey.Passphrase,
			PassphraseSignature: modelKey.PassphraseSignature,
			Fingerprint:         modelKey.Fingerprint,
			Active:              modelKey.Active,
		}
	}
	return keys
}

// Helper functions for safe type handling
func safeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func safeStringWithDefault(val string, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}

func safeInt(val int64) int {
	return int(val)
}

func safeIntWithDefault(val int64, defaultVal int) int {
	if val == 0 {
		return defaultVal
	}
	return int(val)
}

// Determine if user is in active subscription
func isActiveSubscription(plans []models.UserPlan) bool {
	now := time.Now().Unix()

	for _, plan := range plans {
		if plan.Status == "active" &&
			plan.CurrentPeriodStart <= now &&
			(plan.CurrentPeriodEnd == 0 || plan.CurrentPeriodEnd > now) {
			return true
		}
	}

	return false
}

// Determine if user account is delinquent
func isDelinquentAccount(plans []models.UserPlan) bool {
	for _, plan := range plans {
		if plan.Status == "active" && plan.Delinquent {
			return true
		}
	}

	return false
}

// Determine if user's billing is active
func isActiveBilling(billing *models.UserBilling) bool {
	if billing == nil {
		return false
	}

	return billing.Status == "active" || billing.Status == "paid"
}

// Calculate retention eligibility
func calculateRetentionFlag(user *models.User, plans []models.UserPlan) RetentionFlag {
	flag := RetentionFlag{
		Eligible: false,
		Reason:   "",
	}

	// If user has been active for more than 3 months
	threeMonthsAgo := time.Now().AddDate(0, -3, 0).Unix()
	if user.CreatedAt < threeMonthsAgo {
		flag.Eligible = true
	}

	// Check if user has active paid plan
	hasActivePaidPlan := false
	for _, plan := range plans {
		if plan.Status == "active" &&
			plan.PlanType != "free" &&
			plan.CurrentPeriodEnd > time.Now().Unix() {
			hasActivePaidPlan = true
			break
		}
	}

	if !hasActivePaidPlan {
		flag.Eligible = false
		flag.Reason = "No active paid plan"
	}

	return flag
}

// Calculate onboarding flags
func calculateOnboardingFlags(ctx context.Context, user *models.User, driveService *drive.Service) OnboardingFlags {
	// Default all to false
	flags := OnboardingFlags{
		DriveSetup: DriveSetup{
			Completed: false,
		},
		RootFolder: RootFolder{
			Completed: false,
		},
		PlanSelection: PlanSelection{
			Completed: false,
		},
	}

	// Check if user has any drive volumes directly from the database
	volumeCount, _ := driveService.CountUserVolumes(ctx, user.ID)
	if volumeCount {
		flags.DriveSetup.Completed = true
	}

	// Check if user has created any drive items
	driveItemsCount, _ := driveService.CountDriveItems(ctx, user.ID)
	if driveItemsCount {
		flags.RootFolder.Completed = true
	}

	// If user has selected a plan
	if len(user.Plans) > 0 {
		flags.PlanSelection.Completed = true
	}

	return flags
}
