package user

import (
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/session"
	"cirrussync-api/internal/user"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/status"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler handles user requests
type Handler struct {
	userService *user.Service
	logger      *logger.Logger
}

// NewHandler creates a new user handler
func NewHandler(userService *user.Service, log *logger.Logger) *Handler {
	return &Handler{
		userService: userService,
		logger:      log,
	}
}

// secureLog logs errors without sensitive data that might expose code or credentials
func (h *Handler) secureLog(err error, message string, route string) {
	// Generate request ID internally
	requestID := utils.GenerateShortID()
	// Log only necessary information, avoid including stack traces or request bodies
	h.logger.WithFields(logrus.Fields{
		"requestID": requestID,
		"route":     route,
		"errorMsg":  err.Error(),
	}).Error(message)
}

// GetUser handles retrieving a user's profile
func (h *Handler) GetUser(c *gin.Context) {
	// Get and validate user ID from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		h.secureLog(err, err.Error(), "getUser")
		c.JSON(http.StatusUnauthorized, NewErrorResponse(err.Error(), status.StatusUnauthorized))
		return
	}

	// Get user from service - returns our custom User type directly
	responseUser, err := h.userService.GetUser(context.Background(), userID)
	if err != nil {
		h.secureLog(err, err.Error(), "getUser")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Return user profile
	c.JSON(http.StatusOK, NewSuccessResponse("", responseUser, status.StatusOK))
}

// UpdateProfile handles updating a user's profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "updateProfile")
		c.JSON(http.StatusUnprocessableEntity, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Get and validate user ID from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		h.secureLog(err, err.Error(), "updateProfile")
		c.JSON(http.StatusUnauthorized, NewErrorResponse(err.Error(), status.StatusUnauthorized))
		return
	}

	// Create updates map
	updates := make(map[string]interface{})
	if req.DisplayName != "" {
		updates["DisplayName"] = req.DisplayName
	}
	if req.PhoneNumber != "" {
		updates["PhoneNumber"] = req.PhoneNumber
	}
	if req.CompanyName != "" {
		updates["CompanyName"] = req.CompanyName
	}

	// Update user
	_, err = h.userService.UpdateUser(context.Background(), userID, updates)
	if err != nil {
		h.secureLog(err, err.Error(), "updateProfile")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Get updated user with all calculated fields
	updatedUser, err := h.userService.GetUser(context.Background(), userID)
	if err != nil {
		h.secureLog(err, err.Error(), "updateProfile")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Return updated user profile
	c.JSON(http.StatusOK, NewSuccessResponse("Profile updated successfully", updatedUser, status.StatusOK))
}

// AddUserKey handles adding a new key for a user
func (h *Handler) AddUserKey(c *gin.Context) {
	var req AddKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "addKey")
		c.JSON(http.StatusUnprocessableEntity, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Get and validate user ID from context
	userID, err := h.getUserIDFromContext(c)
	if err != nil {
		h.secureLog(err, err.Error(), "addKey")
		c.JSON(http.StatusUnauthorized, NewErrorResponse(err.Error(), status.StatusUnauthorized))
		return
	}

	// Create user key
	key := user.UserKey{
		PublicKey:           req.PublicKey,
		PrivateKey:          req.PrivateKey,
		Passphrase:          req.Passphrase,
		PassphraseSignature: req.PassphraseSignature,
		Fingerprint:         req.Fingerprint,
		Version:             int(req.Version),
	}

	// Add key
	_, err = h.userService.AddUserKey(context.Background(), userID, key)
	if err != nil {
		h.secureLog(err, err.Error(), "addKey")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Get updated user with all calculated fields
	updatedUser, err := h.userService.GetUser(context.Background(), userID)
	if err != nil {
		h.secureLog(err, err.Error(), "addKey")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Return updated user profile
	c.JSON(http.StatusOK, NewSuccessResponse("Key added successfully", updatedUser, status.StatusOK))
}

// Helper function to extract and validate user ID from context
func (h *Handler) getUserIDFromContext(c *gin.Context) (string, error) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		return "", session.ErrSessionNotFound
	}
	userID, ok := userIDInterface.(string)
	if !ok {
		return "", session.ErrInvalidInput
	}
	return userID, nil
}
