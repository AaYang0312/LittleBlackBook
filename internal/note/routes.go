package note

import (
	"xbs/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, jwtSecret string) {
	rg.GET("/notes/latest", h.Latest)
	rg.GET("/notes/:id", h.Detail)
	rg.GET("/users/:id/notes", h.ListByUser)
	auth := rg.Group("", middleware.JWTAuth(jwtSecret))
	auth.POST("/notes", h.Publish)
	auth.POST("/notes/images", h.UploadImage)
	auth.DELETE("/notes/:id", h.Delete)
}
