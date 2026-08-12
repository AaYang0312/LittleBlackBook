package feed

import (
	"xbs/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, jwtSecret string) {
	auth := rg.Group("", middleware.JWTAuth(jwtSecret))
	auth.GET("/feed/following", h.Following)
}
