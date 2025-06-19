package drive

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"cirrussync-api/internal/drive"
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/user"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/status"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Default constants for pagination and timeouts
const (
	defaultLimit     = 100
	maxLimit         = 500
	defaultOffset    = 0
	defaultTimeout   = 10 * time.Second
	extendedTimeout  = 15 * time.Second
	defaultSortBy    = "createdAt"
	defaultSortDir   = "asc"
	readPermission   = "user-read"
	writePermission  = "user-write"
	userIDContextKey = "userID"
	scopesContextKey = "scopes"
)

// Handler handles drive API requests
type Handler struct {
	driveService *drive.Service
	userService  *user.Service
	logger       *logger.Logger
}

// NewHandler creates a new drive handler
func NewHandler(driveService *drive.Service, userService *user.Service, log *logger.Logger) *Handler {
	return &Handler{
		driveService: driveService,
		userService:  userService,
		logger:       log,
	}
}

// secureLog logs errors without sensitive data that might expose code or credentials
func (h *Handler) secureLog(err error, message, route string) {
	// Generate request ID internally
	requestID := utils.GenerateShortID()
	// Log only necessary information, avoid including stack traces or request bodies
	h.logger.WithFields(logrus.Fields{
		"requestID": requestID,
		"route":     route,
		"errorMsg":  err.Error(),
	}).Error(message)
}

// handleServiceError maps service errors to appropriate HTTP responses
func (h *Handler) handleServiceError(err error, route string) (int, int16, string) {
	h.secureLog(err, "Error in "+route, route)

	statusCode := http.StatusInternalServerError
	apiStatus := status.StatusInternalServerError
	message := err.Error()

	switch {
	// Conflict errors
	case errors.Is(err, drive.ErrVolumeAlreadyExists),
		errors.Is(err, drive.ErrRootShareAlreadyExists),
		errors.Is(err, drive.ErrAllocationAlreadyExists),
		errors.Is(err, drive.ErrNameConflict):
		statusCode = http.StatusConflict
		apiStatus = status.StatusConflict

	// Not found errors
	case errors.Is(err, drive.ErrShareNotFound),
		errors.Is(err, drive.ErrUserNotFound),
		errors.Is(err, drive.ErrVolumeNotFound),
		errors.Is(err, drive.ErrFolderNotFound):
		statusCode = http.StatusNotFound
		apiStatus = status.StatusNotFound

	// Permission errors
	case errors.Is(err, drive.ErrUnauthorized),
		errors.Is(err, drive.ErrInsufficientPermissions):
		statusCode = http.StatusForbidden
		apiStatus = status.StatusForbidden

	// Resource limit errors
	case errors.Is(err, drive.ErrStorageQuotaExceeded):
		statusCode = http.StatusPaymentRequired
		apiStatus = status.StatusStorageQuotaExceeded

	// Bad request errors
	case errors.Is(err, drive.ErrNotAFolder):
		statusCode = http.StatusBadRequest
		apiStatus = status.StatusBadRequest

	// Creation errors - keep as internal server errors
	case errors.Is(err, drive.ErrVolumeCreation),
		errors.Is(err, drive.ErrAllocationCreation),
		errors.Is(err, drive.ErrShareCreation),
		errors.Is(err, drive.ErrMembershipCreation),
		errors.Is(err, drive.ErrFolderCreation),
		errors.Is(err, drive.ErrItemRetrieval):
		// These remain as internal server errors
	}

	return statusCode, apiStatus, message
}

// getUserIDAndCheckPermission extracts user ID and checks if they have required permission
// Returns userID, error (nil if successful)
func (h *Handler) getUserIDAndCheckPermission(c *gin.Context, requiredScope string) (string, error) {
	// Get user ID from context (set by auth middleware)
	userIDRaw, exists := c.Get(userIDContextKey)
	if !exists {
		return "", drive.ErrUnauthorized
	}

	userID, ok := userIDRaw.(string)
	if !ok {
		return "", errors.New("invalid user ID format")
	}

	// Check for required permission scope
	scopesRaw, exists := c.Get(scopesContextKey)
	if !exists {
		return "", drive.ErrInsufficientPermissions
	}

	// Convert scopes to string slice
	scopes, ok := scopesRaw.([]string)
	if !ok {
		return "", errors.New("invalid scopes format")
	}

	// Check for required permission
	if slices.Contains(scopes, requiredScope) {
		return userID, nil
	}

	return "", drive.ErrInsufficientPermissions
}

// getPaginationParams extracts and validates pagination parameters with defaults
func (h *Handler) getPaginationParams(c *gin.Context, defaultLimitVal, maxLimitVal int) (int, int) {
	limit := defaultLimitVal
	offset := defaultOffset

	limitParam := c.Query("limit")
	offsetParam := c.Query("offset")

	if limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = min(parsedLimit, maxLimitVal)
		}
	}

	if offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}

// getSortingParams extracts and validates sorting parameters
func (h *Handler) getSortingParams(c *gin.Context, validSortFields map[string]bool) (string, string) {
	sortBy := defaultSortBy
	sortDir := defaultSortDir

	sortByParam := c.Query("sortBy")
	sortDirParam := c.Query("sortDir")

	if sortByParam != "" && validSortFields[sortByParam] {
		sortBy = sortByParam
	}

	if sortDirParam == "desc" {
		sortDir = "desc"
	}

	return sortBy, sortDir
}

// respondWithError sends a standardized error response
func (h *Handler) respondWithError(c *gin.Context, statusCode int, apiStatus int16, message string) {
	c.JSON(statusCode, NewErrorResponse(message, apiStatus))
}

// validateRequestParam validates a required request parameter
func (h *Handler) validateRequestParam(value, name string) error {
	if value == "" {
		return errors.New(name + " is required")
	}
	return nil
}

// CreateDriveVolume handles the creation of a user drive structure
func (h *Handler) CreateDriveVolume(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, writePermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultTimeout)
	defer cancel()

	// Get user details
	user, err := h.userService.GetUserById(ctx, userID)
	if err != nil {
		h.secureLog(err, "Failed to retrieve user", "createDrive")
		h.respondWithError(c, http.StatusInternalServerError, status.StatusInternalServerError, "Failed to retrieve user")
		return
	}

	// Parse request body
	var req CreateDriveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "createDrive")
		c.JSON(http.StatusBadRequest, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Convert request to model
	modelReq := req.ToModelRequest()

	// Call service to create drive structure
	err = h.driveService.CreateDriveStructure(
		ctx,
		user,
		modelReq.DriveVolume,
		modelReq.DriveShare,
		modelReq.DriveShareMembership,
	)

	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "createDrive")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	c.JSON(http.StatusCreated, NewSuccessResponse("Drive structure created successfully", status.StatusCreated))
}

// CreateDriveFolder handles creating a folder under a specific share
func (h *Handler) CreateDriveFolder(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, writePermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Get share ID from URL path
	shareID := c.Param("shareID")
	if err := h.validateRequestParam(shareID, "ShareID"); err != nil {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, err.Error())
		return
	}

	// Parse request body
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "createFolder")
		c.JSON(http.StatusBadRequest, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Convert request to drive item
	folderInput := &models.DriveItem{
		ParentID:                req.ParentId,
		Name:                    req.Name,
		Hash:                    req.Hash,
		SignatureEmail:          req.SignatureEmail,
		NodeKey:                 req.NodeKey,
		NodePassphrase:          req.NodePassphrase,
		NodePassphraseSignature: req.NodePassphraseSignature,
		FolderProperties: &models.FolderProperties{
			NodeHashKey: req.NodeHashKey,
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultTimeout)
	defer cancel()

	// Call service to create folder
	folder, err := h.driveService.CreateDriveFolder(ctx, userID, shareID, folderInput)
	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "createFolder")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	// Return created folder
	c.JSON(http.StatusCreated, NewFolderResponse(folder, status.StatusCreated))
}

// GetUserShares handles the retrieval of shares for a user
func (h *Handler) GetUserShares(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, readPermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Get pagination parameters
	limit, offset := h.getPaginationParams(c, 50, 100)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultTimeout)
	defer cancel()

	// Call service to get shares
	shares, total, err := h.driveService.GetSharesByUserID(ctx, userID, limit, offset)
	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "getUserShares")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	// Return shares
	c.JSON(http.StatusOK, NewSharesListResponse(shares, limit, offset, total, userID, status.StatusOK))
}

// GetShareByID returns a share with all its memberships
func (h *Handler) GetShareByID(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, readPermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Get share ID from URL path
	shareID := c.Param("shareID")
	if err := h.validateRequestParam(shareID, "ShareID"); err != nil {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, err.Error())
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultTimeout)
	defer cancel()

	// Call service to get share with memberships
	share, memberships, err := h.driveService.GetShareWithAllMemberships(ctx, shareID, userID)
	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "getShareById")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	// Return share with memberships
	c.JSON(http.StatusOK, NewShareWithMembershipsResponse(share, memberships, userID, status.StatusOK))
}

// BatchGetShares handles retrieving multiple shares in a single request
func (h *Handler) BatchGetShares(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, readPermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Parse request body
	var req BatchSharesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "batchGetShares")
		c.JSON(http.StatusBadRequest, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Validate request
	if len(req.ShareIDs) == 0 {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, "At least one share ID is required")
		return
	}

	if len(req.ShareIDs) > 50 {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, "Maximum 50 share IDs allowed per request")
		return
	}

	// Create a context with timeout - use extended timeout for batch operations
	ctx, cancel := context.WithTimeout(c.Request.Context(), extendedTimeout)
	defer cancel()

	// Use the BatchGetSharesWithMemberships method from the improved service
	sharesWithMemberships, err := h.driveService.BatchGetSharesWithMemberships(ctx, req.ShareIDs, userID)
	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "batchGetShares")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	// Return batch result
	c.JSON(http.StatusOK, NewBatchSharesResponse(sharesWithMemberships, userID, status.StatusOK))
}

// GetLinkByID handles retrieving any link (file or folder) by its ID
func (h *Handler) GetLinkByID(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, readPermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Get link ID from URL path
	linkID := c.Param("linkID")
	if err := h.validateRequestParam(linkID, "Link ID"); err != nil {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, err.Error())
		return
	}

	// Call service method to get link by ID
	item, err := h.driveService.GetLinkByID(c.Request.Context(), linkID, userID)
	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "getLinkById")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	// Return appropriate response based on item type
	c.JSON(http.StatusOK, NewDriveItemResponse(item, status.StatusOK))
}

// GetFolderContents handles retrieving the contents of a folder
func (h *Handler) GetFolderContents(c *gin.Context) {
	// Check user permissions
	userID, err := h.getUserIDAndCheckPermission(c, readPermission)
	if err != nil {
		h.handlePermissionError(c, err)
		return
	}

	// Get share ID from URL path
	shareID := c.Param("shareID")
	if err := h.validateRequestParam(shareID, "ShareID"); err != nil {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, err.Error())
		return
	}

	// Get folder ID from URL path
	folderID := c.Param("folderID")
	if err := h.validateRequestParam(folderID, "FolderID"); err != nil {
		h.respondWithError(c, http.StatusBadRequest, status.StatusBadRequest, err.Error())
		return
	}

	// Get pagination parameters
	limit, offset := h.getPaginationParams(c, defaultLimit, maxLimit)

	// Get sorting parameters
	validSortFields := map[string]bool{
		"createdAt":  true,
		"modifiedAt": true,
		"size":       true,
		"type":       true,
	}
	sortBy, sortDir := h.getSortingParams(c, validSortFields)

	// Call service method to get folder contents
	items, total, err := h.driveService.GetFolderContents(
		c.Request.Context(),
		shareID,
		folderID,
		userID,
		limit,
		offset,
		sortBy,
		sortDir,
	)

	if err != nil {
		statusCode, apiStatus, message := h.handleServiceError(err, "getFolderContents")
		h.respondWithError(c, statusCode, apiStatus, message)
		return
	}

	// Return folder contents
	c.JSON(http.StatusOK, NewFolderContentsResponse(items, limit, offset, total, sortBy, sortDir, status.StatusOK))
}

// Helper method to handle permission errors
func (h *Handler) handlePermissionError(c *gin.Context, err error) {
	var message string
	var statusCode int

	switch {
	case errors.Is(err, drive.ErrUnauthorized):
		message = "User not authenticated"
		statusCode = http.StatusUnauthorized
	case errors.Is(err, drive.ErrInsufficientPermissions):
		message = "Insufficient permissions"
		statusCode = http.StatusForbidden
	default:
		message = err.Error()
		statusCode = http.StatusForbidden
	}

	h.respondWithError(c, statusCode, status.StatusForbidden, message)
}
