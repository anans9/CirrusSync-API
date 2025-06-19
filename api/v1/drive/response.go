package drive

import (
	"cirrussync-api/internal/drive"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/utils"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// BaseResponse represents the base structure for all API responses
type BaseResponse struct {
	Code   int16  `json:"code"`
	Detail string `json:"detail"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	BaseResponse
	Error string `json:"error,omitempty"`
}

// SuccessResponse represents a simple success message
type SuccessResponse struct {
	BaseResponse
	Message string `json:"message,omitempty"`
}

// DriveResponse represents a response containing drive data
type DriveResponse struct {
	BaseResponse
	Drive interface{} `json:"drive"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code int16) ErrorResponse {
	return ErrorResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Error with requestId " + utils.GenerateShortID(),
		},
		Error: message,
	}
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(message string, code int16) SuccessResponse {
	return SuccessResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Message: message,
	}
}

// NewDriveResponse creates a new drive data response
func NewDriveResponse(drive interface{}, code int16) DriveResponse {
	return DriveResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Drive: drive,
	}
}

// NewValidationError creates a validation error response
func NewValidationError(err error, code int16) ErrorResponse {
	if errs, ok := err.(validator.ValidationErrors); ok && len(errs) > 0 {
		full := errs[0].Error()
		parts := strings.SplitN(full, "Error:", 2)
		message := full
		if len(parts) == 2 {
			message = strings.TrimSpace(parts[1])
		}
		return NewErrorResponse(message, code)
	}
	return NewErrorResponse("Invalid request format", code)
}

// FileProperties represents file-specific properties
type FileProperties struct {
	ContentHash         string         `json:"contentHash"`
	ContentKeyPacket    string         `json:"contentKeyPacket"`
	ContentKeySignature string         `json:"contentKeySignature"`
	ActiveRevision      ActiveRevision `json:"activeRevision"`
}

// FolderProperties represents folder-specific properties
type FolderProperties struct {
	NodeHashKey       string `json:"nodeHashKey"`
	NodeHashSignature string `json:"nodeHashSignature"`
}

// SharingDetails represents sharing information
type SharingDetails struct {
	SharedUrl  string `json:"sharedUrl"`
	ShareID    string `json:"shareId"`
	SharedWith string `json:"sharedWith"`
	SharedAt   string `json:"sharedAt"`
	SharedBy   string `json:"sharedBy"`
}

// ThumbnailURLInfo represents thumbnail URL information
type ThumbnailURLInfo struct {
	BareURL string `json:"bareUrl"`
	Token   string `json:"token"`
}

// ActiveRevision represents the active revision of a file
type ActiveRevision struct {
	ID                   string           `json:"id"`
	CreatedAt            int64            `json:"createdAt"`
	Size                 int64            `json:"size"`
	State                int              `json:"state"`
	Thumbnail            int              `json:"thumbnail"`
	ThumbnailDownloadUrl string           `json:"thumbnailDownloadUrl"`
	ThumbnailURLInfo     ThumbnailURLInfo `json:"thumbnailUrlInfo"`
	SignatureEmail       string           `json:"signatureEmail"`
	ThumbnailSignature   string           `json:"thumbnailSignature"`
}

// DriveItemResponseData represents a unified drive item in the response
// Can represent both files and folders
type DriveItemResponseData struct {
	ID                      string            `json:"id"`
	ParentId                *string           `json:"parentId"`
	ShareId                 string            `json:"shareId"`
	VolumeId                string            `json:"volumeId"`
	Type                    int               `json:"type"`
	Name                    string            `json:"name"`
	Hash                    string            `json:"hash"`
	NameSignatureEmail      *string           `json:"nameSignatureEmail"`
	State                   int               `json:"state"`
	Size                    int64             `json:"size"`
	TotalSize               int64             `json:"totalSize,omitempty"`
	MimeType                *string           `json:"mimeType"`
	Attributes              *string           `json:"attributes"`
	NodeKey                 string            `json:"nodeKey"`
	NodePassphrase          string            `json:"nodePassphrase"`
	NodePassphraseSignature string            `json:"nodePassphraseSignature"`
	SignatureEmail          string            `json:"signatureEmail"`
	CreatedAt               int64             `json:"createdAt"`
	ModifiedAt              int64             `json:"modifiedAt"`
	IsTrashed               bool              `json:"isTrashed"`
	Permissions             int               `json:"permissions"`
	PermissionExpiresAt     *int64            `json:"permissionExpiresAt,omitempty"`
	ExpirationTime          *int64            `json:"expirationTime,omitempty"`
	IsShared                bool              `json:"isShared"`
	SharingDetails          *SharingDetails   `json:"sharingDetails,omitempty"`
	ShareUrls               *[]string         `json:"shareUrls,omitempty"`
	ShareIDs                *[]string         `json:"shareIDs,omitempty"`
	ShareCount              *int              `json:"shareCount,omitempty"`
	NbUrls                  *int              `json:"nbUrls,omitempty"`
	UrlsExpired             *int              `json:"urlsExpired,omitempty"`
	FileProperties          *FileProperties   `json:"fileProperties,omitempty"`
	FolderProperties        *FolderProperties `json:"folderProperties,omitempty"`
	XAttr                   *string           `json:"xattr,omitempty"`
}

// FileResponse represents a response containing a file
type FileResponse struct {
	BaseResponse
	File *DriveItemResponseData `json:"file"`
}

// FolderResponse represents a response containing a folder
type FolderResponse struct {
	BaseResponse
	Folder *DriveItemResponseData `json:"folder"`
}

// NewDriveItemResponse creates a response based on the item type
func NewDriveItemResponse(item *models.DriveItem, code int16) interface{} {
	if item == nil {
		return ErrorResponse{
			BaseResponse: BaseResponse{
				Code:   code,
				Detail: "Error with requestId " + utils.GenerateShortID(),
			},
			Error: "Item not found",
		}
	}

	// Check item type and return appropriate response
	switch item.Type {
	case 1: // Folder
		return FolderResponse{
			BaseResponse: BaseResponse{
				Code:   code,
				Detail: "Success with requestId " + utils.GenerateShortID(),
			},
			Folder: convertToDriveItemResponseData(item),
		}
	case 2: // File
		return FileResponse{
			BaseResponse: BaseResponse{
				Code:   code,
				Detail: "Success with requestId " + utils.GenerateShortID(),
			},
			File: convertToDriveItemResponseData(item),
		}
	default: // Any other type
		return DriveResponse{
			BaseResponse: BaseResponse{
				Code:   code,
				Detail: "Success with requestId " + utils.GenerateShortID(),
			},
			Drive: convertToDriveItemResponseData(item),
		}
	}
}

// For backward compatibility
func NewFolderResponse(folder *models.DriveItem, code int16) FolderResponse {
	return FolderResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Folder: convertToDriveItemResponseData(folder),
	}
}

// Response for folder contents
type FolderContentsResponse struct {
	BaseResponse
	Items      []*DriveItemResponseData `json:"items"`
	Pagination PaginationData           `json:"pagination"`
}

// PaginationData represents pagination information
type PaginationData struct {
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	TotalItems int     `json:"totalItems"`
	SortBy     *string `json:"sortBy,omitempty"`
	SortDir    *string `json:"sortDir,omitempty"`
}

// NewFolderContentsResponse creates a new folder contents response
func NewFolderContentsResponse(items []*models.DriveItem, limit, offset, total int, sortBy, sortDir string, code int16) FolderContentsResponse {
	// For small datasets, process sequentially to avoid goroutine overhead
	if len(items) < 50 {
		responseItems := make([]*DriveItemResponseData, len(items))
		for i, item := range items {
			responseItems[i] = convertToDriveItemResponseData(item)
		}

		return FolderContentsResponse{
			BaseResponse: BaseResponse{
				Code:   code,
				Detail: "Success with requestId " + utils.GenerateShortID(),
			},
			Items: responseItems,
			Pagination: PaginationData{
				Limit:      limit,
				Offset:     offset,
				TotalItems: total,
				SortBy:     &sortBy,
				SortDir:    &sortDir,
			},
		}
	}

	// For larger datasets, use parallel processing
	resultChan := make(chan struct {
		index int
		item  *DriveItemResponseData
	}, len(items))

	// Use a wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(len(items))

	// Process items in parallel for large datasets
	for i, item := range items {
		go func(idx int, itm *models.DriveItem) {
			defer wg.Done()
			resultChan <- struct {
				index int
				item  *DriveItemResponseData
			}{
				index: idx,
				item:  convertToDriveItemResponseData(itm),
			}
		}(i, item)
	}

	// Close the channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in order
	responseItems := make([]*DriveItemResponseData, len(items))
	for result := range resultChan {
		responseItems[result.index] = result.item
	}

	return FolderContentsResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Items: responseItems,
		Pagination: PaginationData{
			Limit:      limit,
			Offset:     offset,
			TotalItems: total,
			SortBy:     &sortBy,
			SortDir:    &sortDir,
		},
	}
}

// Helper function to convert DriveItem to DriveItemResponseData
func convertToDriveItemResponseData(item *models.DriveItem) *DriveItemResponseData {
	if item == nil {
		return nil
	}

	// Create base response with common fields for both types
	responseData := &DriveItemResponseData{
		ID:                      item.ID,
		ParentId:                item.ParentID,
		ShareId:                 item.ShareID,
		VolumeId:                item.VolumeID,
		Type:                    item.Type,
		State:                   item.State,
		Name:                    item.Name,
		MimeType:                item.MimeType,
		Hash:                    item.Hash,
		Size:                    item.Size,
		SignatureEmail:          item.SignatureEmail,
		NodeKey:                 item.NodeKey,
		NodePassphrase:          item.NodePassphrase,
		NodePassphraseSignature: item.NodePassphraseSignature,
		CreatedAt:               item.CreatedAt,
		ModifiedAt:              item.ModifiedAt,
		Permissions:             item.Permissions,
		IsTrashed:               item.IsTrashed,
		IsShared:                item.IsShared,
	}

	// Add type-specific properties based on the item type
	switch item.Type {
	case 1: // Folder
		if item.FolderProperties != nil {
			responseData.FolderProperties = &FolderProperties{
				NodeHashKey: item.FolderProperties.NodeHashKey,
				// Add NodeHashSignature if needed
			}
		}
	case 2: // File
		if item.FileProperties != nil {
			responseData.FileProperties = &FileProperties{
				ContentHash:         item.FileProperties.ContentHash,
				ContentKeyPacket:    item.FileProperties.ContentKeyPacket,
				ContentKeySignature: item.FileProperties.ContentKeySignature,
				// Add ActiveRevision if needed
			}
		}
	}

	return responseData
}

// MembershipResponseData represents a membership in the response
type MembershipResponseData struct {
	ID                  string `json:"id"`
	ShareId             string `json:"shareId"`
	UserId              string `json:"userId"`
	MemberId            string `json:"memberId"`
	Inviter             string `json:"inviter"`
	Permissions         int    `json:"permissions"`
	KeyPacket           string `json:"keyPacket,omitempty"`
	KeyPacketSignature  string `json:"keyPacketSignature,omitempty"`
	SessionKeySignature string `json:"sessionKeySignature,omitempty"`
	State               int    `json:"state"`
	CreatedAt           int64  `json:"createdAt"`
	ModifiedAt          int64  `json:"modifiedAt"`
	CanUnlock           *bool  `json:"canUnlock"`
}

// convertToMembershipResponseData converts a DriveShareMembership model to response data
func convertToMembershipResponseData(membership *models.DriveShareMembership) *MembershipResponseData {
	if membership == nil {
		return nil
	}

	canUnlock := membership.CanUnlock
	if membership.CanUnlock != nil {
		canUnlock = membership.CanUnlock
	}

	return &MembershipResponseData{
		ID:                  membership.ID,
		ShareId:             membership.ShareID,
		UserId:              membership.UserID,
		MemberId:            membership.MemberID,
		Inviter:             membership.Inviter,
		Permissions:         membership.Permissions,
		KeyPacket:           membership.KeyPacket,
		KeyPacketSignature:  membership.KeyPacketSignature,
		SessionKeySignature: membership.SessionKeySignature,
		State:               membership.State,
		CreatedAt:           membership.CreatedAt,
		ModifiedAt:          membership.ModifiedAt,
		CanUnlock:           canUnlock,
	}
}

// ShareResponseData represents a share in the response
type ShareResponseData struct {
	ID                       string                    `json:"id"`
	VolumeId                 string                    `json:"volumeId"`
	UserId                   string                    `json:"userId"`
	Type                     int                       `json:"type"`
	State                    int                       `json:"state"`
	Creator                  string                    `json:"creator"`
	Locked                   bool                      `json:"locked"`
	CreatedAt                int64                     `json:"createdAt"`
	ModifiedAt               int64                     `json:"modifiedAt"`
	LinkId                   string                    `json:"linkId"`
	PermissionsMask          int                       `json:"permissionsMask"`
	BlockSize                int64                     `json:"blockSize"`
	VolumeSoftDeleted        bool                      `json:"volumeSoftDeleted"`
	ShareKey                 string                    `json:"shareKey,omitempty"`
	SharePassphrase          string                    `json:"sharePassphrase,omitempty"`
	SharePassphraseSignature string                    `json:"sharePassphraseSignature,omitempty"`
	IsOwner                  bool                      `json:"isOwner"`
	Memberships              []*MembershipResponseData `json:"memberships,omitempty"`
}

// SharesListResponse represents a response for a list of shares
type SharesListResponse struct {
	BaseResponse
	Shares     []*ShareResponseData `json:"shares"`
	Pagination PaginationData       `json:"pagination"`
}

// NewSharesListResponse creates a response for a list of shares
func NewSharesListResponse(shares []*models.DriveShare, limit, offset, total int, userID string, code int16) SharesListResponse {
	responseShares := make([]*ShareResponseData, 0, len(shares))

	for _, share := range shares {
		if share == nil {
			continue
		}

		// Convert share to response data
		responseShare := convertToShareResponseData(share)

		// Set isOwner correctly based on userID
		responseShare.IsOwner = share.UserID == userID

		responseShares = append(responseShares, responseShare)
	}

	return SharesListResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Shares: responseShares,
		Pagination: PaginationData{
			Limit:      limit,
			Offset:     offset,
			TotalItems: total,
		},
	}
}

// convertToShareResponseData converts a DriveShare model to response data
func convertToShareResponseData(share *models.DriveShare) *ShareResponseData {
	if share == nil {
		return nil
	}

	return &ShareResponseData{
		ID:                       share.ID,
		VolumeId:                 share.VolumeID,
		UserId:                   share.UserID,
		Type:                     share.Type,
		State:                    share.State,
		Creator:                  share.Creator,
		Locked:                   share.Locked,
		CreatedAt:                share.CreatedAt,
		ModifiedAt:               share.ModifiedAt,
		LinkId:                   share.LinkID,
		PermissionsMask:          share.PermissionsMask,
		BlockSize:                share.BlockSize,
		VolumeSoftDeleted:        share.VolumeSoftDeleted,
		ShareKey:                 share.ShareKey,
		SharePassphrase:          share.SharePassphrase,
		SharePassphraseSignature: share.SharePassphraseSignature,
		IsOwner:                  false, // Will be set based on current user
	}
}

// ShareWithMembershipsResponse represents a response with a share and its memberships
type ShareWithMembershipsResponse struct {
	BaseResponse
	Share *ShareResponseData `json:"share"`
}

// NewShareWithMembershipsResponse creates a response with a share and its memberships
func NewShareWithMembershipsResponse(share *models.DriveShare, memberships []*models.DriveShareMembership, userID string, code int16) ShareWithMembershipsResponse {
	// Convert memberships directly without duplicate checking
	responseMembers := make([]*MembershipResponseData, 0, len(memberships))

	for _, membership := range memberships {
		if membership == nil {
			continue
		}
		membershipData := convertToMembershipResponseData(membership)
		responseMembers = append(responseMembers, membershipData)
	}

	// Convert share to response data
	shareData := convertToShareResponseData(share)

	// Set IsOwner based on current user
	shareData.IsOwner = share.UserID == userID

	// Attach memberships to the share
	shareData.Memberships = responseMembers

	return ShareWithMembershipsResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Share: shareData,
	}
}

// BatchSharesRequest is the request structure for batch retrieving shares
type BatchSharesRequest struct {
	ShareIDs []string `json:"shareIds" binding:"required,min=1,max=50"`
}

// BatchSharesResponse represents a response for multiple shares with their memberships
type BatchSharesResponse struct {
	BaseResponse
	Shares map[string]*ShareResponseData `json:"shares"`
	Count  int                           `json:"count"`
}

// NewBatchSharesResponse creates a response for multiple shares with their memberships
func NewBatchSharesResponse(sharesWithMemberships map[string]*drive.ShareWithMemberships, userID string, code int16) BatchSharesResponse {
	// Convert to response format
	responseShares := make(map[string]*ShareResponseData)

	for shareID, shareWithMemberships := range sharesWithMemberships {
		if shareWithMemberships == nil || shareWithMemberships.Share == nil {
			continue
		}

		// Convert memberships without duplicate checking
		responseMembers := make([]*MembershipResponseData, 0, len(shareWithMemberships.Memberships))

		for _, membership := range shareWithMemberships.Memberships {
			if membership == nil {
				continue
			}
			membershipData := convertToMembershipResponseData(membership)
			responseMembers = append(responseMembers, membershipData)
		}

		// Convert share to response data
		shareData := convertToShareResponseData(shareWithMemberships.Share)

		// Set IsOwner based on current user
		shareData.IsOwner = shareWithMemberships.Share.UserID == userID

		// Attach memberships to the share
		shareData.Memberships = responseMembers

		responseShares[shareID] = shareData
	}

	return BatchSharesResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Shares: responseShares,
		Count:  len(responseShares),
	}
}
