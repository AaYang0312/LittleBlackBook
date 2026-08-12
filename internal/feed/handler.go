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

// Following 关注页 Feed
// @Summary 关注页 Feed
// @Tags feed
// @Produce json
// @Security BearerAuth
// @Param offset query int false "偏移量"
// @Param size query int false "每页条数"
// @Success 200 {object} response.body
// @Router /feed/following [get]
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
