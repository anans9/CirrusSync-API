package drive

import (
	"github.com/gin-gonic/gin"
)

func RegisterProtectedRoutes(r *gin.RouterGroup, h *Handler) {
	driveGroup := r.Group("")
	driveGroup.POST("/volumes/create", h.CreateDriveVolume)
	driveGroup.POST("/shares/:shareID/folders/create", h.CreateDriveFolder)
	driveGroup.GET("/shares", h.GetUserShares)
	driveGroup.GET("/shares/:shareID", h.GetShareByID)
	driveGroup.GET("/shares/:shareID/links/:linkID", h.GetLinkByID)
	driveGroup.GET("/shares/:shareID/folders/:folderID/children", h.GetFolderContents)
	driveGroup.GET("/shares/:shareID/links/:linkID/rename", h.GetFolderContents)
}
