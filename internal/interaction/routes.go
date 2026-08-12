package interaction

import (
	"xbs/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, jwtSecret string) {
	auth := rg.Group("", middleware.JWTAuth(jwtSecret))
	auth.POST("/users/:id/follow", h.Follow)
	auth.DELETE("/users/:id/follow", h.Unfollow)
	auth.POST("/notes/:id/like", h.Like)
	auth.DELETE("/notes/:id/like", h.Unlike)
	auth.POST("/notes/:id/collect", h.Collect)
	auth.DELETE("/notes/:id/collect", h.Uncollect)
	auth.POST("/notes/:id/comments", h.CreateComment)
	auth.GET("/notes/:id/comments", h.ListComments)
	rg.POST("/internal/rebuild-counts", h.RebuildCounts)
}
