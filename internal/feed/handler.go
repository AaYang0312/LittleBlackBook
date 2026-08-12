package feed

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

func (h *Handler) Following(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, err := h.svc.Inbox(c.Request.Context(), middleware.CurrentUserID(c), offset, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, list)
}
