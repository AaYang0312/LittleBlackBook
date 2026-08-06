package user

import (
	"xbs/internal/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, jwtSecret string) {
	rg.POST("/users/register", h.Register)
	rg.POST("/users/login", h.Login)
	auth := rg.Group("", middleware.JWTAuth(jwtSecret))
	auth.GET("users/me", h.Me)
}
