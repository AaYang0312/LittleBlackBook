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

// Follow 关注用户
// @Summary 关注用户
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标用户ID"
// @Success 200 {object} response.body
// @Router /users/{id}/follow [post]
func (h *Handler) Follow(c *gin.Context) {
	target, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Follow(c.Request.Context(), middleware.CurrentUserID(c), target); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
// Unfollow 取消关注
// @Summary 取消关注
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标用户ID"
// @Success 200 {object} response.body
// @Router /users/{id}/follow [delete]
func (h *Handler) Unfollow(c *gin.Context) {
	target, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Unfollow(c.Request.Context(), middleware.CurrentUserID(c), target); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Like 点赞笔记
// @Summary 点赞
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Success 200 {object} response.body
// @Router /notes/{id}/like [post]
func (h *Handler) Like(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Like(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
// Unlike 取消点赞
// @Summary 取消点赞
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Success 200 {object} response.body
// @Router /notes/{id}/like [delete]
func (h *Handler) Unlike(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Unlike(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
// Collect 收藏笔记
// @Summary 收藏笔记
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Success 200 {object} response.body
// @Router /notes/{id}/collect [post]
func (h *Handler) Collect(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Collect(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
// Uncollect 取消收藏
// @Summary 取消收藏
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Success 200 {object} response.body
// @Router /notes/{id}/collect [delete]
func (h *Handler) Uncollect(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Uncollect(c.Request.Context(), middleware.CurrentUserID(c), noteID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

type commentReq struct {
	Content  string `json:"content" binding:"required"`
	ParentID int64  `json:"parent_id"`
	ReplyTo  int64  `json:"reply_to"`
}

// CreateComment 评论笔记
// @Summary 评论笔记
// @Tags interaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Param request body commentReq true "评论内容"
// @Success 200 {object} response.body
// @Router /notes/{id}/comments [post]
func (h *Handler) CreateComment(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req commentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errs.ErrParam)
		return
	}
	cm, err := h.svc.CreateComment(c.Request.Context(), middleware.CurrentUserID(c), noteID, req.Content, req.ParentID, req.ReplyTo)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cm)
}

// ListComments 评论列表
// @Summary 评论列表（游标分页）
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Param cursor query int false "游标"
// @Param size query int false "每页条数"
// @Success 200 {object} response.body
// @Router /notes/{id}/comments [get]
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
// ListReplies 展开回复
// @Summary 评论回复列表（游标分页，时间正序）
// @Tags interaction
// @Produce json
// @Security BearerAuth
// @Param id path int true "笔记ID"
// @Param cid path int true "顶级评论ID"
// @Param cursor query int false "游标（已加载最后一条回复id）"
// @Param size query int false "每页条数"
// @Success 200 {object} response.body
// @Router /notes/{id}/comments/{cid}/replies [get]
func (h *Handler) ListReplies(c *gin.Context) {
	noteID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	cs, err := h.svc.ListReplies(c.Request.Context(), noteID, cid, cursor, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cs)
}

// RebuildCounts 重建计数
// @Summary 重建计数（内网接口）
// @Tags interaction
// @Produce json
// @Success 200 {object} response.body
// @Router /internal/rebuild-counts [post]
func (h *Handler) RebuildCounts(c *gin.Context) {
	if err := h.svc.RebuildCounts(c.Request.Context()); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
