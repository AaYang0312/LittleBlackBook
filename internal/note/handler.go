// Publish 发布笔记
// @Summary 发布笔记
// @Tags note
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body publishReq true "笔记内容"
// @Success 200 {object} response.body
// @Router /notes [post]
package note

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/middleware"
	"xbs/internal/pkg/response"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

type publishReq struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content"`
	Images  []string `json:"images" binding:"required,min=1"`
}

func (h *Handler) Publish(c *gin.Context) {
	var req publishReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errs.ErrParam)
		return
	}
	n, err := h.svc.Publish(c.Request.Context(), middleware.CurrentUserID(c), req.Title, req.Content, req.Images)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, n)
}

func (h *Handler) Detail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	dto, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto)
}

func (h *Handler) Latest(c *gin.Context) {
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	p, err := h.svc.Latest(c.Request.Context(), cursor, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

// ListByUser 查看某用户的笔记列表
// @Summary 某用户的笔记列表
// @Tags note
// @Produce json
// @Param id path int true "用户ID"
// @Param cursor query int false "游标（上一页最后一条 note id）"
// @Param size query int false "每页数量（默认20，最大50）"
// @Success 200 {object} response.body
// @Router /users/:id/notes [get]
func (h *Handler) ListByUser(c *gin.Context) {
	uid, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	p, err := h.svc.ListByUser(c.Request.Context(), uid, cursor, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *Handler) UploadImage(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, errs.ErrParam)
		return
	}
	src, err := f.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer src.Close()
	url, err := h.svc.UploadImage(c.Request.Context(), middleware.CurrentUserID(c), src, f.Size, f.Filename)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"url": url})
}

func (h *Handler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), id, middleware.CurrentUserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
