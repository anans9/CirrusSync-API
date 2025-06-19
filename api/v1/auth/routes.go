// api/v1/auth/routes.go
package auth

import (
	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(r *gin.RouterGroup, h *Handler) {
	authGroup := r.Group("/auth")

	// Public routes - no authentication required
	authGroup.POST("/login/init", h.HandleLoginInit)
	authGroup.POST("/login/verify", h.HandleLoginVerify)
	authGroup.POST("/signup", h.HandleSignup)
}

// RegisterProtectedRoutes registers all authentication routes
func RegisterProtectedRoutes(r *gin.RouterGroup, h *Handler) {
	authGroup := r.Group("")

	// Private routes - authentication required
	authGroup.POST("/refresh", h.HandleRefreshToken)
	authGroup.POST("/logout", h.HandleLogout)
	authGroup.POST("/change-password", h.HandleChangePassword)
}
