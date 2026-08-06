package user

import (
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/middleware"
	"xbs/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

type registerReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errs.ErrParam)
		return
	}
	u, err := h.svc.Register(c.Request.Context(), req.Username, req.Password, req.Nickname)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, u)
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errs.ErrParam)
		return
	}
	token, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"token": token})
}

func (h *Handler) Me(c *gin.Context) {
	u, err := h.svc.Profile(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, u)
}
