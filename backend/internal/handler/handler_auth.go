package handler

import (
	"net/http"

	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/agridispatch/agridispatch/internal/util"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器。
type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, 42200, err.Error())
		return
	}
	token, err := h.authSvc.Login(req.Username, req.Password)
	if err != nil {
		util.Fail(c, http.StatusUnauthorized, 40100, "invalid username or password")
		return
	}
	util.OK(c, gin.H{"token": token, "username": req.Username})
}

// Me 当前用户。
func (h *AuthHandler) Me(c *gin.Context) {
	util.OK(c, gin.H{"userId": c.GetUint("user_id"), "username": c.GetString("username"), "role": c.GetString("role")})
}
