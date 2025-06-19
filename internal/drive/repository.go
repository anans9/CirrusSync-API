// internal/drive/repository.go
package drive

import (
	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/db"
	"context"
	"errors"
	"sync"

	"gorm.io/gorm"
)

// Repository interface for drive operations
type Repository interface {
	// Basic creation methods
	CreateVolume(ctx context.Context, volume *models.DriveVolume) error
	CreateAllocation(ctx context.Context, allocation *models.VolumeAllocation) error
	CreateShare(ctx context.Context, share *models.DriveShare) error
	CreateMembership(ctx context.Context, membership *models.DriveShareMembership) error
	CreateItem(ctx context.Context, item *models.DriveItem) error

	// Deletion methods
	DeleteVolume(ctx context.Context, volumeID string) error

	// Count methods
	CountDriveItems(ctx context.Context, userID string) (int, error)
	CountUserVolumes(ctx context.Context, userID string) (int, error)
	CountVolumesByUserID(ctx context.Context, userID string) (int, error)
	CountSharesByUserIDAndType(ctx context.Context, userID string, shareType int) (int, error)
	CountAllocationsByUserID(ctx context.Context, userID string) (int, error)
	CountMembershipsByUserID(ctx context.Context, userID string) (int, error)

	// Get methods
	GetShareByID(ctx context.Context, shareID string) (*models.DriveShare, error)
	GetFolderByID(ctx context.Context, folderID string) (*models.DriveItem, error)
	GetLinkByID(ctx context.Context, linkID string) (*models.DriveItem, error)
	GetMembershipByShareAndUserID(ctx context.Context, shareID, userID string) (*models.DriveShareMembership, error)
	GetAllocationByUserID(ctx context.Context, userID string) (*models.VolumeAllocation, error)
	GetVolumeByUserID(ctx context.Context, userID string) (*models.DriveVolume, error)
	GetRootFolderByShareID(ctx context.Context, shareID string) (*models.DriveItem, error)

	// Update methods
	UpdateAllocation(ctx context.Context, allocation *models.VolumeAllocation) error

	// Get collections
	GetFolderContents(ctx context.Context, folderID string) ([]*models.DriveItem, error)
	GetSharesByUserID(ctx context.Context, userID string) ([]*models.DriveShare, error)
	GetSharesByMemberID(ctx context.Context, userID string) ([]*models.DriveShare, error)
	GetMembershipsByShareID(ctx context.Context, shareID string) ([]*models.DriveShareMembership, error)
	GetFolderContentsPaginated(
		ctx context.Context,
		folderID string,
		limit,
		offset int,
		sortBy,
		sortDir string,
	) ([]*models.DriveItem, int, error)

	// Batch operations
	BatchGetMembershipsByShareIDs(ctx context.Context, shareIDs []string) (map[string][]*models.DriveShareMembership, error)
	BatchGetSharesByIDs(ctx context.Context, shareIDs []string) (map[string]*models.DriveShare, error)
	BatchGetFoldersByIDs(ctx context.Context, folderIDs []string) (map[string]*models.DriveItem, error)
}

// repo implements the Repository interface
type repo struct {
	db             *gorm.DB
	volumeRepo     db.Repository[models.DriveVolume]
	allocationRepo db.Repository[models.VolumeAllocation]
	shareRepo      db.Repository[models.DriveShare]
	membershipRepo db.Repository[models.DriveShareMembership]
	itemRepo       db.Repository[models.DriveItem]
	mutex          sync.Mutex // For operations that need synchronization
}

// NewRepository creates a new drive repository
func NewRepository(database *gorm.DB) Repository {
	return &repo{
		db:             database,
		volumeRepo:     db.NewRepositoryWithDB[models.DriveVolume](database),
		allocationRepo: db.NewRepositoryWithDB[models.VolumeAllocation](database),
		shareRepo:      db.NewRepositoryWithDB[models.DriveShare](database),
		membershipRepo: db.NewRepositoryWithDB[models.DriveShareMembership](database),
		itemRepo:       db.NewRepositoryWithDB[models.DriveItem](database),
	}
}

// CreateVolume creates a new drive volume
func (r *repo) CreateVolume(ctx context.Context, volume *models.DriveVolume) error {
	return r.volumeRepo.Create(ctx, volume)
}

// CreateAllocation creates a new volume allocation
func (r *repo) CreateAllocation(ctx context.Context, allocation *models.VolumeAllocation) error {
	return r.allocationRepo.Create(ctx, allocation)
}

// CreateShare creates a new drive share
func (r *repo) CreateShare(ctx context.Context, share *models.DriveShare) error {
	return r.shareRepo.Create(ctx, share)
}

// CreateMembership creates a new drive membership
func (r *repo) CreateMembership(ctx context.Context, membership *models.DriveShareMembership) error {
	return r.membershipRepo.Create(ctx, membership)
}

// CreateItem creates a new drive item
func (r *repo) CreateItem(ctx context.Context, item *models.DriveItem) error {
	return r.itemRepo.Create(ctx, item)
}

// DeleteVolume deletes a volume by ID
func (r *repo) DeleteVolume(ctx context.Context, volumeID string) error {
	return r.volumeRepo.Delete(ctx, volumeID)
}

// CountDriveItems counts the number of drive items for a user
func (r *repo) CountDriveItems(ctx context.Context, userID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DriveItem{}).
		Joins("JOIN drive_shares ON drive_items.share_id = drive_shares.id").
		Where("drive_shares.user_id = ?", userID).
		Count(&count).Error

	return int(count), err
}

// CountUserVolumes counts volumes and allocations for a user with parallel queries
func (r *repo) CountUserVolumes(ctx context.Context, userID string) (int, error) {
	var volumeCount, allocationCount int64

	// Use a waitgroup to run both queries in parallel
	var wg sync.WaitGroup
	var volumeErr, allocationErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		err := r.db.WithContext(ctx).
			Model(&models.DriveVolume{}).
			Where("user_id = ?", userID).
			Count(&volumeCount).Error
		volumeErr = err
	}()

	go func() {
		defer wg.Done()
		err := r.db.WithContext(ctx).
			Model(&models.VolumeAllocation{}).
			Where("user_id = ?", userID).
			Count(&allocationCount).Error
		allocationErr = err
	}()

	wg.Wait()

	if volumeErr != nil {
		return 0, volumeErr
	}

	if allocationErr != nil {
		return 0, allocationErr
	}

	return int(volumeCount + allocationCount), nil
}

// CountVolumesByUserID counts the number of volumes for a user
func (r *repo) CountVolumesByUserID(ctx context.Context, userID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DriveVolume{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return int(count), err
}

// CountSharesByUserIDAndType counts the number of shares of a specific type for a user
func (r *repo) CountSharesByUserIDAndType(ctx context.Context, userID string, shareType int) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DriveShare{}).
		Where("user_id = ? AND type = ?", userID, shareType).
		Count(&count).Error

	return int(count), err
}

// CountAllocationsByUserID counts the number of allocations for a user
func (r *repo) CountAllocationsByUserID(ctx context.Context, userID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.VolumeAllocation{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return int(count), err
}

// CountMembershipsByUserID counts the number of memberships for a user
func (r *repo) CountMembershipsByUserID(ctx context.Context, userID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DriveShareMembership{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return int(count), err
}

// GetFolderContents retrieves all items within a folder
func (r *repo) GetFolderContents(ctx context.Context, folderID string) ([]*models.DriveItem, error) {
	var items []models.DriveItem
	err := r.db.WithContext(ctx).
		Where("parent_id = ? AND is_trashed = ?", folderID, false).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*models.DriveItem, len(items))
	for i := range items {
		result[i] = &items[i]
	}

	return result, nil
}

// GetShareByID retrieves a drive share by its ID
func (r *repo) GetShareByID(ctx context.Context, shareID string) (*models.DriveShare, error) {
	var share models.DriveShare
	err := r.db.WithContext(ctx).
		Where("id = ?", shareID).
		First(&share).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}
	return &share, nil
}

// GetMembershipByShareAndUserID retrieves a user's membership for a specific share
func (r *repo) GetMembershipByShareAndUserID(ctx context.Context, shareID, userID string) (*models.DriveShareMembership, error) {
	var membership models.DriveShareMembership
	err := r.db.WithContext(ctx).
		Where("share_id = ? AND user_id = ?", shareID, userID).
		First(&membership).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMembershipNotFound
		}
		return nil, err
	}
	return &membership, nil
}

// GetAllocationByUserID retrieves a user's volume allocation
func (r *repo) GetAllocationByUserID(ctx context.Context, userID string) (*models.VolumeAllocation, error) {
	var allocation models.VolumeAllocation
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND active = ?", userID, true).
		First(&allocation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAllocationNotFound
		}
		return nil, err
	}

	// Fix for the allocated size issue
	if allocation.AllocatedSize <= 0 {
		// Query the correct allocated size from the database by updating the struct directly
		err = r.db.WithContext(ctx).
			Model(&allocation).
			Select("allocated_size").
			Where("id = ?", allocation.ID).
			First(&allocation).Error

		if err != nil {
			return nil, err
		}
	}

	return &allocation, nil
}

// GetVolumeByUserID retrieves a user's drive volume
func (r *repo) GetVolumeByUserID(ctx context.Context, userID string) (*models.DriveVolume, error) {
	var volume models.DriveVolume
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&volume).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVolumeNotFound
		}
		return nil, err
	}
	return &volume, nil
}

// UpdateAllocation updates a volume allocation with proper handling of allocated size
func (r *repo) UpdateAllocation(ctx context.Context, allocation *models.VolumeAllocation) error {
	return r.allocationRepo.Update(ctx, allocation)
}

// GetSharesByUserID retrieves shares owned by a user with optimized query
func (r *repo) GetSharesByUserID(ctx context.Context, userID string) ([]*models.DriveShare, error) {
	var shares []models.DriveShare

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND state = ?", userID, 1). // State 1 = active
		Find(&shares).Error

	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*models.DriveShare, len(shares))
	for i := range shares {
		result[i] = &shares[i]
	}

	return result, nil
}

// GetSharesByMemberID retrieves shares that a user is a member of with optimized query
func (r *repo) GetSharesByMemberID(ctx context.Context, userID string) ([]*models.DriveShare, error) {
	var memberships []models.DriveShareMembership

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND state = ?", userID, 1). // State 1 = active
		Find(&memberships).Error

	if err != nil {
		return nil, err
	}

	if len(memberships) == 0 {
		return []*models.DriveShare{}, nil
	}

	// Extract share IDs
	shareIDs := make([]string, len(memberships))
	for i, m := range memberships {
		shareIDs[i] = m.ShareID
	}

	// Get shares for these IDs
	var shares []models.DriveShare
	err = r.db.WithContext(ctx).
		Where("id IN ? AND state = ?", shareIDs, 1). // State 1 = active
		Find(&shares).Error

	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*models.DriveShare, len(shares))
	for i := range shares {
		result[i] = &shares[i]
	}

	return result, nil
}

// GetMembershipsByShareID retrieves all active memberships for a specific share
func (r *repo) GetMembershipsByShareID(ctx context.Context, shareID string) ([]*models.DriveShareMembership, error) {
	var memberships []models.DriveShareMembership

	err := r.db.WithContext(ctx).
		Where("share_id = ? AND state = ?", shareID, 1). // State 1 = active
		Find(&memberships).Error

	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*models.DriveShareMembership, len(memberships))
	for i := range memberships {
		result[i] = &memberships[i]
	}

	return result, nil
}

// BatchGetMembershipsByShareIDs efficiently retrieves memberships for multiple shares
func (r *repo) BatchGetMembershipsByShareIDs(ctx context.Context, shareIDs []string) (map[string][]*models.DriveShareMembership, error) {
	if len(shareIDs) == 0 {
		return make(map[string][]*models.DriveShareMembership), nil
	}

	var memberships []models.DriveShareMembership

	err := r.db.WithContext(ctx).
		Where("share_id IN ? AND state = ?", shareIDs, 1). // State 1 = active
		Find(&memberships).Error

	if err != nil {
		return nil, err
	}

	// Group memberships by share ID
	result := make(map[string][]*models.DriveShareMembership)

	for i := range memberships {
		shareID := memberships[i].ShareID
		result[shareID] = append(result[shareID], &memberships[i])
	}

	return result, nil
}

// BatchGetSharesByIDs efficiently retrieves multiple shares by IDs
func (r *repo) BatchGetSharesByIDs(ctx context.Context, shareIDs []string) (map[string]*models.DriveShare, error) {
	if len(shareIDs) == 0 {
		return make(map[string]*models.DriveShare), nil
	}

	var shares []models.DriveShare

	err := r.db.WithContext(ctx).
		Where("id IN ? AND state = ?", shareIDs, 1). // State 1 = active
		Find(&shares).Error

	if err != nil {
		return nil, err
	}

	// Map shares by ID for quick lookup
	result := make(map[string]*models.DriveShare, len(shares))

	for i := range shares {
		result[shares[i].ID] = &shares[i]
	}

	return result, nil
}

// BatchGetFoldersByIDs efficiently retrieves multiple folders by IDs
func (r *repo) BatchGetFoldersByIDs(ctx context.Context, folderIDs []string) (map[string]*models.DriveItem, error) {
	if len(folderIDs) == 0 {
		return make(map[string]*models.DriveItem), nil
	}

	var folders []models.DriveItem

	err := r.db.WithContext(ctx).
		Where("id IN ? AND type = ? AND is_trashed = ?", folderIDs, 1, false).
		Find(&folders).Error

	if err != nil {
		return nil, err
	}

	// Map folders by ID for quick lookup
	result := make(map[string]*models.DriveItem, len(folders))

	for i := range folders {
		result[folders[i].ID] = &folders[i]
	}

	return result, nil
}

// Repository function to get a link by ID
func (r *repo) GetLinkByID(ctx context.Context, linkID string) (*models.DriveItem, error) {
	// Find the drive item
	item, err := r.itemRepo.FindByID(ctx, linkID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	return item, nil
}

// Repository function to get a folder by ID
func (r *repo) GetFolderByID(ctx context.Context, folderID string) (*models.DriveItem, error) {
	// Find the drive item
	item, err := r.itemRepo.FindByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}

	return item, nil
}

// Repository function to get folder contents with pagination and sorting
func (r *repo) GetFolderContentsPaginated(
	ctx context.Context,
	folderID string,
	limit,
	offset int,
	sortBy,
	sortDir string,
) ([]*models.DriveItem, int, error) {
	var wg sync.WaitGroup
	var total int64
	var items []models.DriveItem
	var countErr, queryErr error

	resultChan := make(chan struct{}, 2)

	// Map API sort fields to DB column names
	columnMap := map[string]string{
		"createdAt":  "created_at",
		"modifiedAt": "modified_at",
		"size":       "size",
		"type":       "type",
	}

	sqlColumn, exists := columnMap[sortBy]
	if !exists {
		sqlColumn = "created_at"
	}

	// Count total items
	wg.Add(1)
	go func() {
		defer wg.Done()
		countQuery := r.itemRepo.DB().WithContext(ctx).
			Model(&models.DriveItem{}).
			Where("parent_id = ? AND is_trashed = ?", folderID, false)

		if err := countQuery.Count(&total).Error; err != nil {
			countErr = err
		}
		resultChan <- struct{}{}
	}()

	// Query paginated items
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := r.itemRepo.DB().WithContext(ctx).
			Model(&models.DriveItem{}).
			Where("parent_id = ? AND is_trashed = ?", folderID, false)

		// Stable sorting: folder-first + column + id
		if sortDir == "desc" {
			query = query.Order("type DESC").Order(sqlColumn + " DESC").Order("id ASC")
		} else {
			query = query.Order("type DESC").Order(sqlColumn + " ASC").Order("id ASC")
		}

		// Proper offset logic: offset as page number
		query = query.Offset(offset * limit).Limit(limit)

		if err := query.Find(&items).Error; err != nil {
			queryErr = err
		}
		resultChan <- struct{}{}
	}()

	// Wait for both goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for range resultChan {
		select {
		case <-resultChan:
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		}
	}

	if countErr != nil {
		return nil, 0, countErr
	}
	if queryErr != nil {
		return nil, 0, queryErr
	}

	// Convert to []*DriveItem
	result := make([]*models.DriveItem, len(items))
	for i := range items {
		result[i] = &items[i]
	}

	return result, int(total), nil
}

// Repository function to get root folder for a share
func (r *repo) GetRootFolderByShareID(ctx context.Context, shareID string) (*models.DriveItem, error) {
	// Find the root folder for this share (parent_id IS NULL)
	var rootFolder models.DriveItem
	err := r.itemRepo.DB().WithContext(ctx).
		Where("share_id = ? AND type = ? AND parent_id IS NULL", shareID, 1).
		First(&rootFolder).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}

	return &rootFolder, nil
}
