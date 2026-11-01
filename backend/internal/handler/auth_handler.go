package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/dto"
	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/service"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

// AuthHandler is the HTTP boundary of auth: bind, call the service, translate
// the result. No business logic lives here.
type AuthHandler struct {
	service      *service.AuthService
	cookieSecure bool
	cookieMaxAge int
}

func NewAuthHandler(svc *service.AuthService, cookieSecure bool, ttl time.Duration) *AuthHandler {
	return &AuthHandler{
		service:      svc,
		cookieSecure: cookieSecure,
		cookieMaxAge: int(ttl.Seconds()),
	}
}

func toUserResponse(user *model.User) dto.UserResponse {
	return dto.UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.CredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.Fail(c, apperrors.BindError(err))
		return
	}

	user, token, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		apperrors.Fail(c, err)
		return
	}

	h.setSessionCookie(c, token)
	c.JSON(http.StatusCreated, dto.AuthResponse{User: toUserResponse(user)})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.CredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.Fail(c, apperrors.BindError(err))
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		apperrors.Fail(c, err)
		return
	}

	h.setSessionCookie(c, token)
	c.JSON(http.StatusOK, dto.AuthResponse{User: toUserResponse(user)})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := util.UserID(c)
	if !ok {
		apperrors.Fail(c, apperrors.Unauthorized("authentication required"))
		return
	}

	user, err := h.service.UserByID(c.Request.Context(), userID)
	if err != nil {
		apperrors.Fail(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{User: toUserResponse(user)})
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(util.SessionCookieName, token, h.cookieMaxAge, "/", "", h.cookieSecure, true)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(util.SessionCookieName, "", -1, "/", "", h.cookieSecure, true)
}
