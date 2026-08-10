package interaction

import (
	"xbs/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, jwtSecert string) {
	auth := rg.Group("", middleware.JWTAuth(jwtSecert))
	auth.POST("/users/:id/follow", h.Follow)
	auth.DELETE("/users/:id/follow", h.Unfollow)
	auth.POST("/notes/:id/like", h.Like)
	auth.DELETE("/notes/:id/like", h.Unlike)
	auth.POST("/notes/:id/collect", h.Collect)
	auth.DELETE("/notes/:id/collect", h.Uncollect)
}
