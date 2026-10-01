package userauthhttp

import (
	"context"
	"errors"

	"github.com/dujiao-next/internal/constants"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	oidcauthapp "github.com/dujiao-next/internal/modules/identity/oidcauth/application"
	userpresenter "github.com/dujiao-next/internal/modules/identity/userauth/transport/presenter"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

var ErrOIDCUnbindRequiresLocalLogin = errors.New("oidc unbind requires local login method")

// UserOIDCService 是通用 OIDC 端点所需的最小端口。
type UserOIDCService interface {
	StartOIDC(ctx context.Context, intent string, userID uint) (string, error)
	LoginWithOIDC(ctx context.Context, code, state string) (*AuthLoginResult, error)
	BindOIDC(ctx context.Context, userID uint, code, state string) (*externalidentitydomain.Identity, error)
	GetOIDCBinding(ctx context.Context, userID uint) (canUnbind bool, identity *externalidentitydomain.Identity, err error)
	UnbindOIDC(ctx context.Context, userID uint) error
}

// UserOIDCHandler 处理通用 OIDC 登录与绑定 HTTP 请求。
type UserOIDCHandler struct {
	service  UserOIDCService
	recorder LoginRecorder
}

func NewUserOIDCHandler(service UserOIDCService, recorder LoginRecorder) *UserOIDCHandler {
	if service == nil {
		panic("user oidc handler: service is nil")
	}
	return &UserOIDCHandler{service: service, recorder: recorder}
}

type oidcCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

func respondOIDCError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oidcauthapp.ErrOIDCAuthDisabled):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_auth_disabled", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCAuthConfigInvalid):
		ginutil.RespondError(c, response.CodeInternal, "error.oidc_auth_config_invalid", err)
	case errors.Is(err, oidcauthapp.ErrOIDCStateInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_state_invalid", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCDiscoveryFailed):
		ginutil.RespondError(c, response.CodeInternal, "error.oidc_discovery_failed", err)
	case errors.Is(err, oidcauthapp.ErrOIDCTokenExchange):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_token_exchange_failed", err)
	case errors.Is(err, oidcauthapp.ErrOIDCIDTokenInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_id_token_invalid", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCPayloadInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_payload_invalid", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCReplay):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_auth_replayed", nil)
	case errors.Is(err, ErrUserOAuthIdentityExists):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_bind_conflict", nil)
	case errors.Is(err, ErrUserOAuthAlreadyBound):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_already_bound", nil)
	case errors.Is(err, ErrOIDCUnbindRequiresLocalLogin):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_unbind_requires_local_login", nil)
	case errors.Is(err, ErrUserDisabled):
		ginutil.RespondError(c, response.CodeUnauthorized, "error.user_disabled", nil)
	case errors.Is(err, ErrRegistrationDisabled):
		ginutil.RespondError(c, response.CodeForbidden, "error.registration_disabled", nil)
	default:
		ginutil.RespondError(c, response.CodeInternal, "error.login_failed", err)
	}
}

func (h *UserOIDCHandler) recordLogin(c *gin.Context, email string, userID uint, status, failReason, source string) {
	if h == nil || h.recorder == nil || c == nil {
		return
	}
	requestID := ""
	if rid, ok := c.Get("request_id"); ok {
		if value, ok := rid.(string); ok {
			requestID = value
		}
	}
	h.recorder.Record(email, userID, status, failReason, source, c.ClientIP(), c.GetHeader("User-Agent"), requestID)
}

// StartOIDCLogin 返回通用 OIDC 授权 URL（登录流程）。
func (h *UserOIDCHandler) StartOIDCLogin(c *gin.Context) {
	authURL, err := h.service.StartOIDC(c.Request.Context(), "login", 0)
	if err != nil {
		respondOIDCError(c, err)
		return
	}
	response.Success(c, gin.H{"auth_url": authURL})
}

// OIDCLoginCallback 处理通用 OIDC 回调（登录）。
func (h *UserOIDCHandler) OIDCLoginCallback(c *gin.Context) {
	var req oidcCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonBadRequest, constants.LoginLogSourceOIDC)
		ginutil.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	res, err := h.service.LoginWithOIDC(c.Request.Context(), req.Code, req.State)
	if err != nil {
		h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondOIDCError(c, err)
		return
	}
	if res.RequiresTOTP {
		h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, constants.LoginLogPasswordOK2FAPending, constants.LoginLogSourceOIDC)
		response.Success(c, gin.H{
			"requires_totp":        true,
			"challenge_token":      res.ChallengeToken,
			"challenge_expires_at": res.ChallengeExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		})
		return
	}
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceOIDC)
	response.Success(c, gin.H{
		"requires_totp": false,
		"user":          userpresenter.NewUserAuthBriefResp(res.User),
		"token":         res.Token,
		"expires_at":    res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// StartOIDCBind 返回通用 OIDC 授权 URL（绑定流程，需登录）。
func (h *UserOIDCHandler) StartOIDCBind(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}
	authURL, err := h.service.StartOIDC(c.Request.Context(), "bind", uid)
	if err != nil {
		respondOIDCError(c, err)
		return
	}
	response.Success(c, gin.H{"auth_url": authURL})
}

// OIDCBindCallback 处理通用 OIDC 回调（绑定，需登录）。
func (h *UserOIDCHandler) OIDCBindCallback(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}
	var req oidcCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	identity, err := h.service.BindOIDC(c.Request.Context(), uid, req.Code, req.State)
	if err != nil {
		respondOIDCError(c, err)
		return
	}
	response.Success(c, userpresenter.NewOIDCBindingResp(identity))
}

// GetMyOIDCBinding 获取当前用户的通用 OIDC 绑定。
func (h *UserOIDCHandler) GetMyOIDCBinding(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}
	canUnbind, identity, err := h.service.GetOIDCBinding(c.Request.Context(), uid)
	if err != nil {
		respondOIDCError(c, err)
		return
	}
	response.Success(c, userpresenter.NewOIDCBindingResp(identity, canUnbind))
}

// UnbindMyOIDC 解绑当前用户的通用 OIDC。
func (h *UserOIDCHandler) UnbindMyOIDC(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}
	if err := h.service.UnbindOIDC(c.Request.Context(), uid); err != nil {
		respondOIDCError(c, err)
		return
	}
	response.Success(c, gin.H{"unbound": true})
}
