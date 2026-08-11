package interaction

import (
	"strconv"
	"xbs/internal/pkg/errs"
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

func (h *Handler) Like(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Like(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
func (h *Handler) Unlike(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Unlike(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
func (h *Handler) Collect(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Collect(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
func (h *Handler) Uncollect(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Uncollect(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

type commentReq struct {
	Content string `json:"content" binding:"required"`
}

func (h *Handler) CreateComment(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req commentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errs.ErrParam)
		return
	}
	cm, err := h.svc.CreateComment(c.Request.Context(), middleware.CurrentUserID(c), noteID, req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cm)
}

func (h *Handler) ListComments(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	cs, err := h.svc.ListComments(c.Request.Context(), noteID, cursor, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cs)
}
