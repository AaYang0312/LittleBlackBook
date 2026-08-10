package interaction

import (
	"strconv"
	"xbs/internal/pkg/middleware"
	"xbs/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Follow(c *gin.Context) {
	target, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Follow(c.Request.Context(), middleware.CurrentUserID(c), target); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
func (h *Handler) Unfollow(c *gin.Context) {
	target, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Unfollow(c.Request.Context(), middleware.CurrentUserID(c), target); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
