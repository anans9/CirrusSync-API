package csrf

import (
	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(r *gin.RouterGroup, h *Handler) {
	csrfGroup := r.Group("/csrf")
	csrfGroup.GET("", h.HandleCSRFToken)
}
