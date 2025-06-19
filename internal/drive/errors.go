package drive

import "errors"

// Common errors
var (
	ErrUserNotFound       = errors.New("User not found")
	ErrVolumeCreation     = errors.New("Failed to create drive volume")
	ErrAllocationCreation = errors.New("Failed to create volume allocation")
	ErrShareCreation      = errors.New("Failed to create drive share")
	ErrMembershipCreation = errors.New("Failed to create share membership")
	ErrFolderCreation     = errors.New("Failed to create folder")
	ErrItemRetrieval      = errors.New("Failed to retrieve drive items")
	ErrMembershipNotFound = errors.New("Membership not found")
	ErrAllocationNotFound = errors.New("Allocation not found")
	ErrVolumeNotFound     = errors.New("Volume not found")

	ErrVolumeAlreadyExists     = errors.New("User already has a volume")
	ErrRootShareAlreadyExists  = errors.New("User already has a root share")
	ErrAllocationAlreadyExists = errors.New("User already has an allocation")
	ErrMembershipAlreadyExists = errors.New("User already has a share membership")
	ErrNameConflict            = errors.New("A folder with this name already exists in this location")
	ErrShareNotFound           = errors.New("Share not found")
	ErrUnauthorized            = errors.New("You don't have permission to create folders in this share")
	ErrInsufficientPermissions = errors.New("You don't have sufficient permissions for this operation")
	ErrStorageQuotaExceeded    = errors.New("Storage quota exceeded")

	ErrItemNotFound   = errors.New("Link not found")
	ErrFolderNotFound = errors.New("Folder not found")
	ErrNotAFolder     = errors.New("Item is not a folder")
)
