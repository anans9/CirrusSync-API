package auth

import (
	"cirrussync-api/internal/session"

	"github.com/gin-gonic/gin"
)

// GetDeviceDetails extracts the device details from headers
func GetDeviceDetails(c *gin.Context) session.DeviceInfo {
	return session.DeviceInfo{
		ClientName: c.GetHeader("X-Client-Name"),
		AppVersion: c.GetHeader("X-App-Version"),
		ClientUID:  c.GetHeader("X-Client-UID"),
		UserAgent:  c.Request.UserAgent(),
	}
}

// getUserIDFromContext extracts and validates user ID from context
func (h *Handler) getUserIDFromContext(c *gin.Context) (string, error) {
	userIDInterface, exists := c.Get("userId")
	if !exists {
		return "", session.ErrSessionNotFound
	}
	userID, ok := userIDInterface.(string)
	if !ok {
		return "", session.ErrSessionNotFound
	}
	return userID, nil
}

// getSessionIDFromContext extracts and validates session ID from context
func (h *Handler) getSessionIDFromContext(c *gin.Context) (string, error) {
	sessionIDInterface, exists := c.Get("sessionId")
	if !exists {
		return "", session.ErrSessionNotFound
	}
	sessionID, ok := sessionIDInterface.(string)
	if !ok {
		return "", session.ErrSessionNotFound
	}
	return sessionID, nil
}
