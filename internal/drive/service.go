// internal/drive/service.go
package drive

import (
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Permission constants based on Unix-style permissions with additions
const (
	// Basic Unix-style permissions
	EXECUTE_PERMISSION = 1 // Execute/Access: 001
	WRITE_PERMISSION   = 2 // Write: 010
	READ_PERMISSION    = 4 // Read: 100
	RWX_PERMISSIONS    = 7 // Read+Write+Execute: 111

	// Additional custom permissions
	SHARE_PERMISSION = 8  // Permission to share: 1000
	ADMIN_PERMISSION = 16 // Admin permissions: 10000

	// Common permission combinations
	RW_PERMISSIONS  = 6  // Read+Write: 110
	RWS_PERMISSIONS = 14 // Read+Write+Share: 1110
	ALL_PERMISSIONS = 31 // All permissions: 11111 (Read+Write+Execute+Share+Admin)

	// Value 22 = Read(4) + Write(2) + Share(8) + Admin(8) = 22
	MEMBERSHIP_DEFAULT = 22 // Default for share membership

	// Default timeouts
	DEFAULT_TIMEOUT  = 5 * time.Second
	EXTENDED_TIMEOUT = 10 * time.Second
	BATCH_SIZE       = 10 // Process in batches of 10 for parallel operations

	// Cache expiration
	CACHE_EXPIRATION = 1 * time.Hour
)

// NewService creates a new drive service
func NewService(repo Repository, redisClient *redis.Client, logger *logger.Logger) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
		logger:      logger,
	}
}

// CreateDriveStructure sets up the initial drive structure for a new user
func (s *Service) CreateDriveStructure(
	ctx context.Context,
	user *models.User,
	driveVolume *models.DriveVolume,
	driveShare *models.DriveShare,
	driveShareMembership *models.DriveShareMembership,
) error {
	// Check context for cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Create a context with timeout for the operations
	opCtx, cancel := context.WithTimeout(ctx, EXTENDED_TIMEOUT)
	defer cancel()

	// Create result channels for parallel existence checks
	type checkResult struct {
		exists bool
		err    error
	}

	// Run all existence checks in parallel for efficiency
	volumeCheckCh := make(chan checkResult, 1)
	shareCheckCh := make(chan checkResult, 1)
	allocCheckCh := make(chan checkResult, 1)
	memberCheckCh := make(chan checkResult, 1)

	go func() {
		count, err := s.repo.CountVolumesByUserID(opCtx, user.ID)
		volumeCheckCh <- checkResult{exists: count > 0, err: err}
	}()

	go func() {
		count, err := s.repo.CountSharesByUserIDAndType(opCtx, user.ID, 1)
		shareCheckCh <- checkResult{exists: count > 0, err: err}
	}()

	go func() {
		count, err := s.repo.CountAllocationsByUserID(opCtx, user.ID)
		allocCheckCh <- checkResult{exists: count > 0, err: err}
	}()

	go func() {
		count, err := s.repo.CountMembershipsByUserID(opCtx, user.ID)
		memberCheckCh <- checkResult{exists: count > 0, err: err}
	}()

	// Wait for results and check for errors
	volumeCheck := <-volumeCheckCh
	if volumeCheck.err != nil {
		return volumeCheck.err
	}
	if volumeCheck.exists {
		return ErrVolumeAlreadyExists
	}

	shareCheck := <-shareCheckCh
	if shareCheck.err != nil {
		return shareCheck.err
	}
	if shareCheck.exists {
		return ErrRootShareAlreadyExists
	}

	allocCheck := <-allocCheckCh
	if allocCheck.err != nil {
		return allocCheck.err
	}
	if allocCheck.exists {
		return ErrAllocationAlreadyExists
	}

	memberCheck := <-memberCheckCh
	if memberCheck.err != nil {
		return memberCheck.err
	}
	if memberCheck.exists {
		return ErrMembershipAlreadyExists
	}

	// Pre-generate all IDs to ensure consistency
	volumeID := utils.GenerateLinkID()
	allocationID := utils.GenerateLinkID()
	shareID := utils.GenerateLinkID()
	shareLinkID := utils.GenerateLinkID()
	membershipID := utils.GenerateLinkID()

	// Create volume
	volume := &models.DriveVolume{
		ID:       volumeID,
		Name:     driveVolume.Name,
		Hash:     driveVolume.Hash,
		UserID:   user.ID,
		Size:     3221225472, // 3GB default
		PlanType: "free",
	}

	if err := s.repo.CreateVolume(ctx, volume); err != nil {
		return ErrVolumeCreation
	}

	// Create allocation and share in parallel with proper error handling
	type createResult struct {
		err error
	}

	allocCreateCh := make(chan createResult, 1)
	shareCreateCh := make(chan createResult, 1)

	// Create allocation goroutine
	go func() {
		allocation := &models.VolumeAllocation{
			ID:                   allocationID,
			VolumeID:             volume.ID,
			UserID:               user.ID,
			AllocatedSize:        volume.Size,
			UsedSize:             0,
			AllocationPercentage: 100.0,
			Active:               true,
			IsOwner:              true,
		}

		err := s.repo.CreateAllocation(opCtx, allocation)
		allocCreateCh <- createResult{err: err}
	}()

	// Create share goroutine
	go func() {
		share := &models.DriveShare{
			ID:                       shareID,
			VolumeID:                 volume.ID,
			UserID:                   user.ID,
			Type:                     1, // Root share
			State:                    1, // Active
			Creator:                  user.Email,
			LinkID:                   shareLinkID,
			ShareKey:                 driveShare.ShareKey,
			SharePassphrase:          driveShare.SharePassphrase,
			SharePassphraseSignature: driveShare.SharePassphraseSignature,
		}

		err := s.repo.CreateShare(opCtx, share)
		shareCreateCh <- createResult{err: err}
	}()

	// Wait for creation results
	allocResult := <-allocCreateCh
	shareResult := <-shareCreateCh

	// Handle allocation creation error
	if allocResult.err != nil {
		// Cleanup volume since allocation failed
		_ = s.repo.DeleteVolume(ctx, volume.ID)
		return ErrAllocationCreation
	}

	// Handle share creation error
	if shareResult.err != nil {
		// Cleanup volume (allocation will be cascaded)
		_ = s.repo.DeleteVolume(ctx, volume.ID)
		return ErrShareCreation
	}

	// Create share membership
	shareMember := &models.DriveShareMembership{
		ID:                  membershipID,
		ShareID:             shareID,
		UserID:              user.ID,
		MemberID:            user.ID,
		Inviter:             user.Email,
		State:               1,                  // Active
		Permissions:         MEMBERSHIP_DEFAULT, // Full permissions (22)
		KeyPacket:           driveShareMembership.KeyPacket,
		KeyPacketSignature:  driveShareMembership.KeyPacketSignature,
		SessionKeySignature: driveShareMembership.SessionKeySignature,
	}

	if err := s.repo.CreateMembership(ctx, shareMember); err != nil {
		// Cleanup volume (will cascade delete shares and allocations)
		_ = s.repo.DeleteVolume(ctx, volume.ID)
		return ErrMembershipCreation
	}

	// Clear any related cache entries after creating structure
	s.invalidateUserCaches(ctx, user.ID)

	return nil
}

// CheckSharePermissions verifies if a user has specific permissions for a share
func (s *Service) CheckSharePermissions(ctx context.Context, userID, shareID string, requiredPermission int) error {
	// Check context for cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Create cache key for share permissions check
	cacheKey := fmt.Sprintf("perm:%s:%s:%d", userID, shareID, requiredPermission)

	// Try to get from cache first
	var permResult struct {
		HasPermission bool
	}

	err := s.redisClient.GetJSON(ctx, cacheKey, &permResult)
	if err == nil {
		// Found in cache
		if permResult.HasPermission {
			return nil
		}
		return ErrInsufficientPermissions
	}

	// Create channels for parallel operations
	type shareResult struct {
		Share *models.DriveShare
		Err   error
	}

	type membershipResult struct {
		Membership *models.DriveShareMembership
		Err        error
	}

	shareCh := make(chan shareResult, 1)
	membershipCh := make(chan membershipResult, 1)

	// Get share and membership in parallel
	go func() {
		share, err := s.GetShareByID(ctx, shareID)
		shareCh <- shareResult{Share: share, Err: err}
	}()

	go func() {
		membership, err := s.GetMembershipByShareAndUserID(ctx, shareID, userID)
		membershipCh <- membershipResult{Membership: membership, Err: err}
	}()

	// Get share result first
	shareRes := <-shareCh

	if shareRes.Err != nil {
		return ErrShareNotFound
	}

	share := shareRes.Share

	// If user is the owner, they have all permissions
	if share.UserID == userID {
		// Consume membership result to prevent goroutine leak
		<-membershipCh

		// Cache the positive result
		permResult.HasPermission = true
		_ = s.redisClient.SetJSON(ctx, cacheKey, permResult, CACHE_EXPIRATION)

		return nil
	}

	// If share permissions mask is not 0, check if the required permission is allowed
	if share.PermissionsMask != 0 && (share.PermissionsMask&requiredPermission) != requiredPermission {
		// Consume membership result to prevent goroutine leak
		<-membershipCh

		// Cache the negative result
		permResult.HasPermission = false
		_ = s.redisClient.SetJSON(ctx, cacheKey, permResult, CACHE_EXPIRATION)

		return ErrInsufficientPermissions
	}

	// Get membership result
	membershipRes := <-membershipCh

	if membershipRes.Err != nil || membershipRes.Membership == nil {
		// Cache the negative result
		permResult.HasPermission = false
		_ = s.redisClient.SetJSON(ctx, cacheKey, permResult, CACHE_EXPIRATION)

		return ErrUnauthorized
	}

	membership := membershipRes.Membership

	// Check if user has the required permission in their membership
	if (membership.Permissions & requiredPermission) != requiredPermission {
		// Cache the negative result
		permResult.HasPermission = false
		_ = s.redisClient.SetJSON(ctx, cacheKey, permResult, CACHE_EXPIRATION)

		return ErrInsufficientPermissions
	}

	// Cache the positive result
	permResult.HasPermission = true
	_ = s.redisClient.SetJSON(ctx, cacheKey, permResult, CACHE_EXPIRATION)

	return nil
}

// CheckStorageQuota verifies if a user has enough storage space for an operation
func (s *Service) CheckStorageQuota(ctx context.Context, userID string, requiredBytes int64) error {
	// Get user's allocation with caching
	cacheKey := fmt.Sprintf("allocation:%s", userID)

	var allocation models.VolumeAllocation
	err := s.redisClient.GetJSON(ctx, cacheKey, &allocation)

	if err != nil {
		// Cache miss, get from database
		alloc, err := s.repo.GetAllocationByUserID(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to get storage allocation: %w", err)
		}

		allocation = *alloc

		// Cache the allocation
		_ = s.redisClient.SetJSON(ctx, cacheKey, allocation, CACHE_EXPIRATION)
	}

	// Calculate remaining space
	remainingSpace := allocation.AllocatedSize - allocation.UsedSize

	// Check if there's enough space
	if requiredBytes > remainingSpace {
		return ErrStorageQuotaExceeded
	}

	return nil
}

// CreateDriveFolder creates a new folder in the drive with improved parallel execution
func (s *Service) CreateDriveFolder(ctx context.Context, userID string, shareID string, folderInput *models.DriveItem) (*models.DriveItem, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Validate input
	if folderInput == nil {
		return nil, errors.New("folder input cannot be nil")
	}
	if shareID == "" {
		return nil, errors.New("shareID is required")
	}

	// Set the shareID in folderInput
	folderInput.ShareID = shareID

	// Use channels for parallel operations with context support
	type permissionCheckResult struct {
		Share *models.DriveShare
		Err   error
	}

	type quotaCheckResult struct {
		Err error
	}

	type nameCheckResult struct {
		Exists bool
		Err    error
	}

	permissionCh := make(chan permissionCheckResult, 1)
	quotaCh := make(chan quotaCheckResult, 1)
	nameCheckCh := make(chan nameCheckResult, 1)

	// Create a context with timeout for the parallel operations
	opCtx, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
	defer cancel()

	// Check permissions and get share
	go func() {
		select {
		case <-opCtx.Done():
			permissionCh <- permissionCheckResult{nil, opCtx.Err()}
			return
		default:
			// Check if user has write permission
			err := s.CheckSharePermissions(opCtx, userID, shareID, WRITE_PERMISSION)
			if err != nil {
				permissionCh <- permissionCheckResult{nil, err}
				return
			}

			// Get share for volume ID
			share, err := s.GetShareByID(opCtx, shareID)
			permissionCh <- permissionCheckResult{share, err}
		}
	}()

	// Check storage quota
	go func() {
		select {
		case <-opCtx.Done():
			quotaCh <- quotaCheckResult{opCtx.Err()}
			return
		default:
			// For folders, we use a minimal size (just metadata)
			err := s.CheckStorageQuota(opCtx, userID, 1024) // 1KB for folder metadata
			quotaCh <- quotaCheckResult{err}
		}
	}()

	// Check for name conflicts
	go func() {
		select {
		case <-opCtx.Done():
			nameCheckCh <- nameCheckResult{false, opCtx.Err()}
			return
		default:
			if folderInput.ParentID != nil {
				exists, err := s.checkFolderNameExists(opCtx, *folderInput.ParentID, folderInput.Hash)
				nameCheckCh <- nameCheckResult{exists, err}
			} else {
				nameCheckCh <- nameCheckResult{false, nil}
			}
		}
	}()

	// Wait for results from all operations
	permissionResult := <-permissionCh
	quotaResult := <-quotaCh
	nameResult := <-nameCheckCh

	// Check for errors in order of precedence
	if permissionResult.Err != nil {
		return nil, permissionResult.Err
	}
	if quotaResult.Err != nil {
		return nil, quotaResult.Err
	}
	if nameResult.Err != nil {
		return nil, fmt.Errorf("failed to check for name conflicts: %w", nameResult.Err)
	}
	if nameResult.Exists {
		return nil, ErrNameConflict
	}

	// Get the share from the permission check result
	share := permissionResult.Share

	// Generate the initial folderID
	folderID := utils.GenerateLinkID()

	// If ParentID is nil, fetch the share and assign LinkID from share
	if folderInput.ParentID == nil {
		share, err := s.GetShareByID(ctx, shareID)
		if err != nil {
			log.Printf("Error retrieving share with ID %s: %v", shareID, err)
			return nil, ErrShareNotFound
		}

		// Assign the LinkID from the share
		folderID = share.LinkID
	}

	// Create folder with properly initialized structure
	folder := &models.DriveItem{
		ID:                      folderID,
		ParentID:                folderInput.ParentID,
		ShareID:                 shareID,
		VolumeID:                share.VolumeID,
		Type:                    1, // Folder
		Name:                    folderInput.Name,
		Hash:                    folderInput.Hash,
		SignatureEmail:          folderInput.SignatureEmail,
		NodeKey:                 folderInput.NodeKey,
		NodePassphrase:          folderInput.NodePassphrase,
		NodePassphraseSignature: folderInput.NodePassphraseSignature,
		Permissions:             RWX_PERMISSIONS, // Set default folder permissions to 7 (rwx)
	}

	// Initialize folder properties
	folder.FolderProperties = &models.FolderProperties{
		NodeHashKey: folderInput.FolderProperties.NodeHashKey,
	}

	// Create the folder
	if err := s.repo.CreateItem(ctx, folder); err != nil {
		return nil, ErrFolderCreation
	}

	// Update storage used (can be done asynchronously)
	go s.updateStorageUsed(context.Background(), userID, 1024)

	// Invalidate cached parent folder contents
	if folder.ParentID != nil {
		s.invalidateFolderCaches(ctx, *folder.ParentID)
	}

	return folder, nil
}

// updateStorageUsed updates the user's storage usage asynchronously
func (s *Service) updateStorageUsed(ctx context.Context, userID string, bytes int64) {
	// Create a new context with timeout to avoid hanging goroutines
	opCtx, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
	defer cancel()

	// Get from cache first
	cacheKey := fmt.Sprintf("allocation:%s", userID)
	var allocation models.VolumeAllocation
	err := s.redisClient.GetJSON(opCtx, cacheKey, &allocation)

	if err != nil {
		// Cache miss, get from database
		alloc, err := s.repo.GetAllocationByUserID(opCtx, userID)
		if err != nil {
			s.logger.Error("Failed to get allocation for updating storage used", err)
			return
		}
		allocation = *alloc
	}

	allocation.UsedSize += bytes

	// Ensure we don't update with zero AllocatedSize
	if allocation.AllocatedSize <= 0 {
		// Get correct allocation size from database
		alloc, err := s.repo.GetAllocationByUserID(opCtx, userID)
		if err != nil {
			s.logger.Error("Failed to get allocation for updating storage used", err)
			return
		}
		allocation.AllocatedSize = alloc.AllocatedSize
	}

	// Update in database
	err = s.repo.UpdateAllocation(opCtx, &allocation)
	if err != nil {
		s.logger.Error("Failed to update allocation", err)
		return
	}

	// Update in cache
	_ = s.redisClient.SetJSON(opCtx, cacheKey, allocation, CACHE_EXPIRATION)
}

// Helper method to check if a folder with the same name exists
func (s *Service) checkFolderNameExists(ctx context.Context, parentID, folderNameHash string) (bool, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("folder_hash:%s:%s", parentID, folderNameHash)
	var exists bool
	err := s.redisClient.GetJSON(ctx, cacheKey, &exists)
	if err == nil {
		return exists, nil
	}

	// Cache miss, check from database
	items, err := s.repo.GetFolderContents(ctx, parentID)
	if err != nil {
		return false, err
	}

	exists = false
	for _, item := range items {
		if item.Type == 1 && item.Hash == folderNameHash {
			exists = true
			break
		}
	}

	// Cache the result (short expiration as folder contents may change)
	_ = s.redisClient.SetJSON(ctx, cacheKey, exists, 5*time.Minute)

	return exists, nil
}

// CountUserVolumes checks if the user has any drive volumes
func (s *Service) CountUserVolumes(ctx context.Context, userID string) (bool, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Validate input
	if userID == "" {
		return false, errors.New("userID is required")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("volumecount:%s", userID)
	var count int
	err := s.redisClient.GetJSON(ctx, cacheKey, &count)
	if err == nil {
		return count >= 2, nil
	}

	// Cache miss, get from database
	dbCount, err := s.repo.CountUserVolumes(ctx, userID)
	if err != nil {
		return false, ErrItemRetrieval
	}

	// Cache the result
	_ = s.redisClient.SetJSON(ctx, cacheKey, dbCount, CACHE_EXPIRATION)

	return dbCount >= 2, nil
}

// CountDriveItems checks if the user has any drive items
func (s *Service) CountDriveItems(ctx context.Context, userID string) (bool, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Validate input
	if userID == "" {
		return false, errors.New("userID is required")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("itemcount:%s", userID)
	var count int
	err := s.redisClient.GetJSON(ctx, cacheKey, &count)
	if err == nil {
		return count > 0, nil
	}

	// Cache miss, get from database
	dbCount, err := s.repo.CountDriveItems(ctx, userID)
	if err != nil {
		return false, ErrItemRetrieval
	}

	// Cache the result
	_ = s.redisClient.SetJSON(ctx, cacheKey, dbCount, CACHE_EXPIRATION)

	return dbCount > 0, nil
}

// GetSharesByUserID gets all shares for a user with pagination and caching
func (s *Service) GetSharesByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.DriveShare, int, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}

	// Check cache first for total shares
	cacheKey := fmt.Sprintf("shares:%s:all", userID)
	var allShares []*models.DriveShare
	err := s.redisClient.GetJSON(ctx, cacheKey, &allShares)

	if err != nil {
		// Cache miss, fetch from database
		// Create a context with timeout
		opCtx, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
		defer cancel()

		// Create channels for parallel operations
		type sharesResult struct {
			Shares []*models.DriveShare
			Err    error
		}

		ownedSharesCh := make(chan sharesResult, 1)
		memberSharesCh := make(chan sharesResult, 1)

		// Get owned shares in parallel
		go func() {
			shares, err := s.repo.GetSharesByUserID(opCtx, userID)
			ownedSharesCh <- sharesResult{Shares: shares, Err: err}
		}()

		// Get member shares in parallel
		go func() {
			shares, err := s.repo.GetSharesByMemberID(opCtx, userID)
			memberSharesCh <- sharesResult{Shares: shares, Err: err}
		}()

		// Get results
		ownedSharesResult := <-ownedSharesCh
		memberSharesResult := <-memberSharesCh

		// Check for errors
		if ownedSharesResult.Err != nil {
			return nil, 0, fmt.Errorf("failed to get owned shares: %w", ownedSharesResult.Err)
		}

		if memberSharesResult.Err != nil {
			return nil, 0, fmt.Errorf("failed to get member shares: %w", memberSharesResult.Err)
		}

		// Combine results while removing duplicates
		shareMap := make(map[string]*models.DriveShare)

		// Add owned shares to map
		for _, share := range ownedSharesResult.Shares {
			shareMap[share.ID] = share
		}

		// Add member shares to map if not already in map
		for _, share := range memberSharesResult.Shares {
			if _, exists := shareMap[share.ID]; !exists {
				shareMap[share.ID] = share
			}
		}

		// Convert map back to slice
		allShares = make([]*models.DriveShare, 0, len(shareMap))
		for _, share := range shareMap {
			allShares = append(allShares, share)
		}

		// Cache the results
		_ = s.redisClient.SetJSON(ctx, cacheKey, allShares, CACHE_EXPIRATION)
	}

	// Get total count
	total := len(allShares)

	// Apply pagination
	start := offset
	end := offset + limit

	if start >= total {
		return []*models.DriveShare{}, total, nil
	}

	if end > total {
		end = total
	}

	// Return paginated result
	return allShares[start:end], total, nil
}

// GetShareByID retrieves a share with caching
func (s *Service) GetShareByID(ctx context.Context, shareID string) (*models.DriveShare, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check cache first
	cacheKey := fmt.Sprintf("share:%s", shareID)
	var share models.DriveShare
	err := s.redisClient.GetJSON(ctx, cacheKey, &share)
	if err == nil {
		return &share, nil
	}

	// Cache miss, get from database
	dbShare, err := s.repo.GetShareByID(ctx, shareID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	_ = s.redisClient.SetJSON(ctx, cacheKey, dbShare, CACHE_EXPIRATION)

	return dbShare, nil
}

// GetMembershipByShareAndUserID retrieves a membership with caching
func (s *Service) GetMembershipByShareAndUserID(ctx context.Context, shareID, userID string) (*models.DriveShareMembership, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check cache first
	cacheKey := fmt.Sprintf("membership:%s:%s", shareID, userID)
	var membership models.DriveShareMembership
	err := s.redisClient.GetJSON(ctx, cacheKey, &membership)
	if err == nil {
		return &membership, nil
	}

	// Cache miss, get from database
	dbMembership, err := s.repo.GetMembershipByShareAndUserID(ctx, shareID, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	_ = s.redisClient.SetJSON(ctx, cacheKey, dbMembership, CACHE_EXPIRATION)

	return dbMembership, nil
}

// GetShareWithAllMemberships retrieves a share with all its memberships and caching
func (s *Service) GetShareWithAllMemberships(ctx context.Context, shareID, userID string) (*models.DriveShare, []*models.DriveShareMembership, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

	// Check cache first
	cacheKey := fmt.Sprintf("share_with_memberships:%s:%s", shareID, userID)
	var cachedResult struct {
		Share       models.DriveShare
		Memberships []*models.DriveShareMembership
	}

	err := s.redisClient.GetJSON(ctx, cacheKey, &cachedResult)
	if err == nil {
		return &cachedResult.Share, cachedResult.Memberships, nil
	}

	// Cache miss, proceed with parallel database operations
	// Create a context with timeout
	opCtx, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
	defer cancel()

	// Use channels for parallel operations
	type shareResult struct {
		Share *models.DriveShare
		Err   error
	}

	type membershipResult struct {
		Memberships []*models.DriveShareMembership
		Err         error
	}

	type userMembershipResult struct {
		Membership *models.DriveShareMembership
		Err        error
	}

	shareCh := make(chan shareResult, 1)
	membershipsCh := make(chan membershipResult, 1)
	userMembershipCh := make(chan userMembershipResult, 1)

	// Get share in parallel
	go func() {
		share, err := s.GetShareByID(opCtx, shareID)
		shareCh <- shareResult{Share: share, Err: err}
	}()

	// Get all memberships in parallel (will use only if needed)
	go func() {
		memberships, err := s.repo.GetMembershipsByShareID(opCtx, shareID)
		membershipsCh <- membershipResult{Memberships: memberships, Err: err}
	}()

	// Get user membership in parallel
	go func() {
		membership, err := s.GetMembershipByShareAndUserID(opCtx, shareID, userID)
		userMembershipCh <- userMembershipResult{Membership: membership, Err: err}
	}()

	// Get share result first
	shareRes := <-shareCh

	// Check for share errors
	if shareRes.Err != nil {
		// Drain other channels to prevent goroutine leaks
		<-membershipsCh
		<-userMembershipCh
		return nil, nil, ErrShareNotFound
	}

	share := shareRes.Share

	// Check if user is the owner of the share
	isOwner := share.UserID == userID

	// If not the owner, verify the user has a valid membership with permission to view
	if !isOwner {
		userMembershipRes := <-userMembershipCh

		if userMembershipRes.Err != nil || userMembershipRes.Membership == nil {
			// Drain remaining channel
			<-membershipsCh
			return nil, nil, ErrUnauthorized
		}

		userMembership := userMembershipRes.Membership

		// Check if membership is active
		if userMembership.State != 1 {
			// Drain remaining channel
			<-membershipsCh
			return nil, nil, ErrUnauthorized
		}

		// Check if user has at least read permission
		if (userMembership.Permissions & READ_PERMISSION) == 0 {
			// Drain remaining channel
			<-membershipsCh
			return nil, nil, ErrInsufficientPermissions
		}

		// Check if they have sufficient permissions to view other memberships
		hasAdminOrSharePermissions := (userMembership.Permissions & (ADMIN_PERMISSION | SHARE_PERMISSION)) != 0

		if !hasAdminOrSharePermissions {
			// If they don't have admin or share permissions, only return their own membership
			// Drain remaining channel
			<-membershipsCh

			// Cache the result
			cachedResult.Share = *share
			cachedResult.Memberships = []*models.DriveShareMembership{userMembership}
			_ = s.redisClient.SetJSON(ctx, cacheKey, cachedResult, CACHE_EXPIRATION)

			return share, []*models.DriveShareMembership{userMembership}, nil
		}
	} else {
		// Drain user membership channel if owner
		<-userMembershipCh
	}

	// Get all memberships result
	membershipRes := <-membershipsCh

	if membershipRes.Err != nil {
		return nil, nil, fmt.Errorf("failed to get memberships: %w", membershipRes.Err)
	}

	// Cache the result
	cachedResult.Share = *share
	cachedResult.Memberships = membershipRes.Memberships
	_ = s.redisClient.SetJSON(ctx, cacheKey, cachedResult, CACHE_EXPIRATION)

	return share, membershipRes.Memberships, nil
}

// BatchGetSharesWithMemberships with optimized parallelism and caching
func (s *Service) BatchGetSharesWithMemberships(ctx context.Context, shareIDs []string, userID string) (map[string]*ShareWithMemberships, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create a context with timeout
	opCtx, cancel := context.WithTimeout(ctx, EXTENDED_TIMEOUT)
	defer cancel()

	// Create result map with proper capacity
	result := make(map[string]*ShareWithMemberships, len(shareIDs))

	// Create a wait group for each share ID
	var wg sync.WaitGroup
	var resultMutex sync.Mutex

	// Process all share IDs in parallel with a limit on concurrency
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent operations

	for _, shareID := range shareIDs {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check cache first
			cacheKey := fmt.Sprintf("share_with_memberships:%s:%s", id, userID)
			var cachedResult struct {
				Share       models.DriveShare
				Memberships []*models.DriveShareMembership
			}

			err := s.redisClient.GetJSON(opCtx, cacheKey, &cachedResult)
			if err == nil {
				// Cache hit
				resultMutex.Lock()
				result[id] = &ShareWithMemberships{
					Share:       &cachedResult.Share,
					Memberships: cachedResult.Memberships,
				}
				resultMutex.Unlock()
				return
			}

			// Cache miss, get from database
			share, memberships, err := s.GetShareWithAllMemberships(opCtx, id, userID)
			if err != nil {
				// Skip this share on error
				return
			}

			// Add to result
			resultMutex.Lock()
			result[id] = &ShareWithMemberships{
				Share:       share,
				Memberships: memberships,
			}
			resultMutex.Unlock()
		}(shareID)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return result, nil
}

// Helper function to check if user has admin or share permissions
func hasAdminOrSharePermissions(memberships []*models.DriveShareMembership, userID string) bool {
	for _, membership := range memberships {
		if membership.UserID == userID &&
			(membership.Permissions&(ADMIN_PERMISSION|SHARE_PERMISSION)) != 0 {
			return true
		}
	}
	return false
}

// GetLinkByID retrieves any drive item (file or folder) by ID with caching
func (s *Service) GetLinkByID(ctx context.Context, linkID, userID string) (*models.DriveItem, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check cache first
	cacheKey := fmt.Sprintf("link:%s", linkID)
	var item models.DriveItem
	err := s.redisClient.GetJSON(ctx, cacheKey, &item)
	if err == nil {
		// Check permissions separately
		shareID := item.ShareID
		// Check if user is owner or has permissions
		permErr := s.CheckSharePermissions(ctx, userID, shareID, READ_PERMISSION)
		if permErr != nil {
			return nil, permErr
		}
		return &item, nil
	}

	// Cache miss, get with parallel operations
	// Create a context with timeout for parallel operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
	defer cancel()

	// Use channels for parallel operations
	type itemResult struct {
		Item *models.DriveItem
		Err  error
	}
	itemCh := make(chan itemResult, 1)

	// Get item in parallel
	go func() {
		item, err := s.repo.GetLinkByID(ctxWithTimeout, linkID)
		itemCh <- itemResult{Item: item, Err: err}
	}()

	// Get result
	itemRes := <-itemCh

	// Check for errors
	if itemRes.Err != nil {
		return nil, ErrItemNotFound
	}

	item = *itemRes.Item

	// Cache the item
	_ = s.redisClient.SetJSON(ctx, cacheKey, item, CACHE_EXPIRATION)

	// Get the share to check permissions - start in parallel
	type shareResult struct {
		Share *models.DriveShare
		Err   error
	}
	shareCh := make(chan shareResult, 1)
	go func() {
		share, err := s.GetShareByID(ctxWithTimeout, item.ShareID)
		shareCh <- shareResult{Share: share, Err: err}
	}()

	// Get share result
	shareRes := <-shareCh
	if shareRes.Err != nil {
		return nil, ErrShareNotFound
	}
	share := shareRes.Share

	// Check if user is the owner of the share
	isOwner := share.UserID == userID

	// If not the owner, verify the user has a valid membership with permission to view
	if !isOwner {
		// Check permissions through share membership
		permissionCh := make(chan error, 1)
		go func() {
			err := s.CheckSharePermissions(ctxWithTimeout, userID, item.ShareID, READ_PERMISSION)
			permissionCh <- err
		}()
		if err := <-permissionCh; err != nil {
			return nil, err
		}
	}

	// User has permission to view this item
	return &item, nil
}

// GetFolderByID retrieves a folder by ID with caching
func (s *Service) GetFolderByID(ctx context.Context, folderID, userID string) (*models.DriveItem, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check cache first
	cacheKey := fmt.Sprintf("folder:%s", folderID)
	var folder models.DriveItem
	err := s.redisClient.GetJSON(ctx, cacheKey, &folder)
	if err == nil {
		// Check permissions separately
		shareID := folder.ShareID

		// Check if user is owner or has permissions
		permErr := s.CheckSharePermissions(ctx, userID, shareID, READ_PERMISSION)
		if permErr != nil {
			return nil, permErr
		}

		return &folder, nil
	}

	// Cache miss, get with parallel operations
	// Create a context with timeout for parallel operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
	defer cancel()

	// Use channels for parallel operations
	type folderResult struct {
		Folder *models.DriveItem
		Err    error
	}

	folderCh := make(chan folderResult, 1)

	// Get folder in parallel
	go func() {
		folder, err := s.repo.GetFolderByID(ctxWithTimeout, folderID)
		folderCh <- folderResult{Folder: folder, Err: err}
	}()

	// Get result
	folderRes := <-folderCh

	// Check for errors
	if folderRes.Err != nil {
		return nil, ErrFolderNotFound
	}

	folder = *folderRes.Folder

	// Verify that it's actually a folder (Type = 1)
	if folder.Type != 1 {
		return nil, ErrNotAFolder
	}

	// Cache the folder
	_ = s.redisClient.SetJSON(ctx, cacheKey, folder, CACHE_EXPIRATION)

	// Get the share to check permissions - start in parallel
	type shareResult struct {
		Share *models.DriveShare
		Err   error
	}

	shareCh := make(chan shareResult, 1)

	go func() {
		share, err := s.GetShareByID(ctxWithTimeout, folder.ShareID)
		shareCh <- shareResult{Share: share, Err: err}
	}()

	// Get share result
	shareRes := <-shareCh

	if shareRes.Err != nil {
		return nil, ErrShareNotFound
	}

	share := shareRes.Share

	// Check if user is the owner of the share
	isOwner := share.UserID == userID

	// If not the owner, verify the user has a valid membership with permission to view
	if !isOwner {
		// Check permissions through share membership
		permissionCh := make(chan error, 1)

		go func() {
			err := s.CheckSharePermissions(ctxWithTimeout, userID, folder.ShareID, READ_PERMISSION)
			permissionCh <- err
		}()

		if err := <-permissionCh; err != nil {
			return nil, err
		}
	}

	// User has permission to view this folder
	return &folder, nil
}

// GetFolderContents retrieves folder contents with caching for the first page only
func (s *Service) GetFolderContents(
	ctx context.Context,
	shareID,
	folderID,
	userID string,
	limit,
	offset int,
	sortBy,
	sortDir string,
) ([]*models.DriveItem, int, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}

	// Only cache the first page (offset 0)
	// For other pages, go directly to the database
	if offset == 0 && limit <= 100 {
		// We'll cache the first page only to optimize memory usage
		cacheKey := fmt.Sprintf("folder_contents:%s:%s:%s:%d", folderID, sortBy, sortDir, limit)
		var folderContents struct {
			Items []*models.DriveItem
			Total int
		}

		// Try to get from cache
		err := s.redisClient.GetJSON(ctx, cacheKey, &folderContents)
		if err == nil {
			// Check if user has permission to view this folder
			permErr := s.CheckSharePermissions(ctx, userID, shareID, READ_PERMISSION)
			if permErr != nil {
				return nil, 0, permErr
			}

			// Return the cached first page
			return folderContents.Items, folderContents.Total, nil
		}
	}

	// Cache miss or non-first page, proceed with database operations
	// Create a context with timeout for parallel operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, DEFAULT_TIMEOUT)
	defer cancel()

	// Use errgroup for coordinated error handling in parallel operations
	g, gCtx := errgroup.WithContext(ctxWithTimeout)

	var share *models.DriveShare
	var folder *models.DriveItem
	var items []*models.DriveItem
	var total int

	// Get share info in parallel
	g.Go(func() error {
		var err error
		// Verify that share exists
		share, err = s.GetShareByID(gCtx, shareID)
		if err != nil {
			return ErrShareNotFound
		}
		return nil
	})

	// If not empty, verify folderID in parallel
	if folderID != "" {
		g.Go(func() error {
			var err error
			// Verify it's a folder and belongs to the share
			folder, err = s.GetFolderByID(gCtx, folderID, userID)
			if err != nil {
				return ErrFolderNotFound
			}

			// Verify it's a folder
			if folder.Type != 1 {
				return ErrNotAFolder
			}

			// Verify folder belongs to the specified share
			if folder.ShareID != shareID {
				return ErrUnauthorized
			}

			return nil
		})
	}

	// Wait for parallel operations to complete
	if err := g.Wait(); err != nil {
		return nil, 0, err
	}

	// Check if user is the owner of the share
	isOwner := share.UserID == userID

	// If not the owner, verify user has permission in parallel with content fetch
	var permissionErr error
	permDone := make(chan struct{})

	if !isOwner {
		go func() {
			// Check permissions through share membership
			permissionErr = s.CheckSharePermissions(ctx, userID, shareID, READ_PERMISSION)
			close(permDone)
		}()
	} else {
		close(permDone) // Close immediately for owner
	}

	// Start fetching content in parallel - this runs regardless of permission check
	contentDone := make(chan struct{})
	var contentErr error

	go func() {
		// Get folder contents with pagination and sorting
		var err error
		items, total, err = s.repo.GetFolderContentsPaginated(ctx, folderID, limit, offset, sortBy, sortDir)
		if err != nil {
			contentErr = fmt.Errorf("failed to get folder contents: %w", err)
		}
		close(contentDone)
	}()

	// Wait for permission check to complete
	<-permDone
	if !isOwner && permissionErr != nil {
		// Cancel content fetch by waiting but ignoring result
		<-contentDone
		return nil, 0, permissionErr
	}

	// Wait for content fetch to complete
	<-contentDone
	if contentErr != nil {
		return nil, 0, contentErr
	}

	// Cache only the first page results
	if offset == 0 && limit <= 100 {
		cacheKey := fmt.Sprintf("folder_contents:%s:%s:%s:%d", folderID, sortBy, sortDir, limit)
		folderContents := struct {
			Items []*models.DriveItem
			Total int
		}{
			Items: items,
			Total: total,
		}

		// Store in cache with expiration
		_ = s.redisClient.SetJSON(ctx, cacheKey, folderContents, CACHE_EXPIRATION)
	}

	// Return the results directly (no additional pagination needed)
	return items, total, nil
}

// Cache invalidation helpers

// invalidateFolderCaches invalidates caches related to a folder
func (s *Service) invalidateFolderCaches(ctx context.Context, folderID string) {
	// Delete folder contents cache - using pattern to match all sort options
	folderContentsPattern := fmt.Sprintf("folder_contents:%s:*", folderID)
	s.deleteKeysWithPattern(ctx, folderContentsPattern)

	// Delete folder cache
	folderCacheKey := fmt.Sprintf("folder:%s", folderID)
	deleted, err := s.redisClient.Delete(ctx, folderCacheKey)
	if err != nil {
		s.logger.Errorf("Failed to delete folder cache for folder %s: %v", folderID, err)
	} else if deleted {
		s.logger.Debugf("Deleted folder cache for folder %s", folderID)
	}

	// Delete all hash existence caches for this folder
	hashExistencePattern := fmt.Sprintf("folder_hash:%s:*", folderID)
	s.deleteKeysWithPattern(ctx, hashExistencePattern)
}

// invalidateShareCaches invalidates caches related to a share
func (s *Service) invalidateShareCaches(ctx context.Context, shareID string) {
	// Delete share cache
	shareCacheKey := fmt.Sprintf("share:%s", shareID)
	deleted, err := s.redisClient.Delete(ctx, shareCacheKey)
	if err != nil {
		s.logger.Errorf("Failed to delete share cache for share %s: %v", shareID, err)
	} else if deleted {
		s.logger.Debugf("Deleted share cache for share %s", shareID)
	}

	// Delete all share membership caches for this share
	shareWithMembershipsPattern := fmt.Sprintf("share_with_memberships:%s:*", shareID)
	s.deleteKeysWithPattern(ctx, shareWithMembershipsPattern)

	// Delete all permission check caches related to this share
	permissionPattern := fmt.Sprintf("perm:*:%s:*", shareID)
	s.deleteKeysWithPattern(ctx, permissionPattern)
}

// invalidateUserCaches invalidates caches related to a user
func (s *Service) invalidateUserCaches(ctx context.Context, userID string) {
	// Delete user shares cache
	sharesCacheKey := fmt.Sprintf("shares:%s:all", userID)
	deleted, err := s.redisClient.Delete(ctx, sharesCacheKey)
	if err != nil {
		s.logger.Errorf("Failed to delete shares cache for user %s: %v", userID, err)
	} else if deleted {
		s.logger.Debugf("Deleted shares cache for user %s", userID)
	}

	// Delete user allocation cache
	allocCacheKey := fmt.Sprintf("allocation:%s", userID)
	deleted, err = s.redisClient.Delete(ctx, allocCacheKey)
	if err != nil {
		s.logger.Errorf("Failed to delete allocation cache for user %s: %v", userID, err)
	} else if deleted {
		s.logger.Debugf("Deleted allocation cache for user %s", userID)
	}

	// Delete volume count cache
	volumeCountCacheKey := fmt.Sprintf("volumecount:%s", userID)
	deleted, err = s.redisClient.Delete(ctx, volumeCountCacheKey)
	if err != nil {
		s.logger.Errorf("Failed to delete volume count cache for user %s: %v", userID, err)
	} else if deleted {
		s.logger.Debugf("Deleted volume count cache for user %s", userID)
	}

	// Delete item count cache
	itemCountCacheKey := fmt.Sprintf("itemcount:%s", userID)
	deleted, err = s.redisClient.Delete(ctx, itemCountCacheKey)
	if err != nil {
		s.logger.Errorf("Failed to delete item count cache for user %s: %v", userID, err)
	} else if deleted {
		s.logger.Debugf("Deleted item count cache for user %s", userID)
	}

	// Delete all permission caches related to this user
	permissionPattern := fmt.Sprintf("perm:%s:*", userID)
	s.deleteKeysWithPattern(ctx, permissionPattern)

	// Delete all share membership caches related to this user
	membershipPattern := fmt.Sprintf("share_with_memberships:*:%s", userID)
	s.deleteKeysWithPattern(ctx, membershipPattern)
}

// deleteKeysWithPattern deletes all keys matching a pattern using SCAN
func (s *Service) deleteKeysWithPattern(ctx context.Context, pattern string) {
	// Execute Lua script to find and delete all keys matching the pattern
	script := `
		local cursor = "0"
		local keyCount = 0
		repeat
			local result = redis.call("SCAN", cursor, "MATCH", ARGV[1], "COUNT", 100)
			cursor = result[1]
			local keys = result[2]
			if #keys > 0 then
				keyCount = keyCount + #keys
				redis.call("DEL", unpack(keys))
			end
		until cursor == "0"
		return keyCount
	`

	// Run the script
	deletedCount, err := s.redisClient.Eval(ctx, script, []string{}, []string{pattern})
	if err != nil {
		s.logger.Errorf("Failed to delete keys with pattern %s: %v", pattern, err)
	} else {
		count, ok := deletedCount.(int64)
		if ok && count > 0 {
			s.logger.Debugf("Deleted %d keys matching pattern %s", count, pattern)
		}
	}
}
