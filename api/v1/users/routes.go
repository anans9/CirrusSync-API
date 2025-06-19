package user

import (
	"github.com/gin-gonic/gin"
)

func RegisterProtectedRoutes(r *gin.RouterGroup, h *Handler) {
	user := r.Group("/")
	user.GET("@me", h.GetUser)
}
