package interaction

import (
	"xbs/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, jwtSecert string) {
	auth := rg.Group("", middleware.JWTAuth(jwtSecert))
	auth.POST("/users/:id/follow", h.Follow)
	auth.DELETE("/users/:id/follow", h.Unfollow)
}
