package mfa

import (
	"github.com/gin-gonic/gin"
)

// RegisterProtectedRoutes registers all MFA routes
func RegisterProtectedRoutes(r *gin.RouterGroup, handler *Handler) {
	mfa := r.Group("/mfa")

	// Email verification routes
	mfa.POST("/email/challenge", handler.HandleSendVerification)
	mfa.GET("/email/verify", handler.HandleVerifyEmail)
	mfa.POST("/email/verify", handler.HandleVerifyEmail)
	mfa.GET("/email/check", handler.HandleCheckVerification)
	mfa.POST("/email/check", handler.HandleCheckVerification)

	// TOTP routes
	mfa.POST("/totp/generate", handler.HandleGenerateTOTP)
	mfa.POST("/totp/verify", handler.HandleVerifyTOTP)
	mfa.POST("/totp/validate", handler.HandleValidateTOTP)
	mfa.POST("/totp/disable", handler.HandleDisableTOTP)
}
