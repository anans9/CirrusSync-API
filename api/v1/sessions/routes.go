package session

import (
	"github.com/gin-gonic/gin"
)

// RegisterProtectedRoutes registers session routes
func RegisterProtectedRoutes(r *gin.RouterGroup, h *Handler) {
	sessionGroup := r.Group("")
	{
		// Get current session
		sessionGroup.GET("/current", h.GetCurrentSession)

		// Invalidate current session (logout)
		sessionGroup.DELETE("/current", h.InvalidateSession)

		// Get all sessions for current user
		sessionGroup.GET("", h.GetAllActiveSessions)

		// Invalidate all sessions for current user
		sessionGroup.DELETE("", h.InvalidateAllSessions)

		// Invalidate a specific session by ID
		sessionGroup.DELETE("/:id", h.InvalidateSessionByID)
	}
}
