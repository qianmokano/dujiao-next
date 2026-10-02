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
	LoginWithOIDCPassword(ctx context.Context, account, password string) (*AuthLoginResult, *OIDCMFAChallengeView, error)
	CompleteOIDCMFA(ctx context.Context, challenge, mfaType, passcode string) (*AuthLoginResult, error)
	SendOIDCRegisterCode(ctx context.Context, email string) error
	RegisterWithOIDC(ctx context.Context, email, password, code, displayName string) (*AuthLoginResult, error)
}

// OIDCMFAChallengeView 是页内登录的 MFA 挑战视图。
type OIDCMFAChallengeView struct {
	Token string            `json:"token"`
	Props []OIDCMFAPropView `json:"props"`
}

// OIDCMFAPropView 是一种可选的二步验证方式。
type OIDCMFAPropView struct {
	MfaType string `json:"mfa_type"`
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
	case errors.Is(err, oidcauthapp.ErrOIDCInvalidCredentials):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_invalid_credentials", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCAccountFrozen):
		ginutil.RespondError(c, response.CodeTooManyRequests, "error.oidc_account_frozen", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCCaptchaRequired):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_captcha_required", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCMFAChallengeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_mfa_challenge_invalid", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCMFACodeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_mfa_code_invalid", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCCodeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_code_invalid", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCEmailExists):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_email_exists", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCCodeResendWait):
		ginutil.RespondError(c, response.CodeBadRequest, "error.oidc_code_resend_wait", nil)
	case errors.Is(err, oidcauthapp.ErrOIDCRemoteRejected):
		ginutil.RespondError(c, response.CodeInternal, "error.oidc_remote_rejected", err)
	case errors.Is(err, ErrUserDisabled):
		ginutil.RespondError(c, response.CodeUnauthorized, "error.user_disabled", nil)
	case errors.Is(err, ErrEmailNotVerified):
		ginutil.RespondError(c, response.CodeForbidden, "error.email_not_verified", nil)
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

type oidcPasswordLoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// OIDCPasswordLogin 页内直连登录(凭据由后端代理交 IdP 校验)。
func (h *UserOIDCHandler) OIDCPasswordLogin(c *gin.Context) {
	var req oidcPasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonBadRequest, constants.LoginLogSourceOIDC)
		ginutil.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	res, challenge, err := h.service.LoginWithOIDCPassword(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondOIDCError(c, err)
		return
	}
	if challenge != nil {
		props := make([]OIDCMFAPropView, 0, len(challenge.Props))
		for _, p := range challenge.Props {
			props = append(props, OIDCMFAPropView{MfaType: p.MfaType})
		}
		h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogPasswordOK2FAPending, constants.LoginLogSourceOIDC)
		response.Success(c, gin.H{
			"requires_mfa": true,
			"mfa_challenge": gin.H{
				"token": challenge.Token,
				"props": props,
			},
		})
		return
	}
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceOIDC)
	response.Success(c, gin.H{
		"requires_mfa":         false,
		"requires_totp":        res.RequiresTOTP,
		"user":                 userpresenter.NewUserAuthBriefResp(res.User),
		"token":                res.Token,
		"expires_at":           res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"challenge_token":      res.ChallengeToken,
		"challenge_expires_at": res.ChallengeExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type oidcMFARequest struct {
	Challenge string `json:"challenge" binding:"required"`
	MfaType   string `json:"mfa_type" binding:"required"`
	Passcode  string `json:"passcode" binding:"required"`
}

// OIDCMFAComplete 续接页内登录的 MFA 二步验证。
func (h *UserOIDCHandler) OIDCMFAComplete(c *gin.Context) {
	var req oidcMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.service.CompleteOIDCMFA(c.Request.Context(), req.Challenge, req.MfaType, req.Passcode)
	if err != nil {
		h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondOIDCError(c, err)
		return
	}
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceOIDC)
	response.Success(c, gin.H{
		"requires_totp":        res.RequiresTOTP,
		"user":                 userpresenter.NewUserAuthBriefResp(res.User),
		"token":                res.Token,
		"expires_at":           res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"challenge_token":      res.ChallengeToken,
		"challenge_expires_at": res.ChallengeExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type oidcRegisterCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

// OIDCRegisterSendCode 页内注册:发送邮箱验证码(经 IdP)。
func (h *UserOIDCHandler) OIDCRegisterSendCode(c *gin.Context) {
	var req oidcRegisterCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	if err := h.service.SendOIDCRegisterCode(c.Request.Context(), req.Email); err != nil {
		respondOIDCError(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}

type oidcRegisterRequest struct {
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Code        string `json:"code" binding:"required"`
	DisplayName string `json:"display_name"`
}

// OIDCRegister 页内注册:IdP 完成邮箱验证码开户并即时登录。
func (h *UserOIDCHandler) OIDCRegister(c *gin.Context) {
	var req oidcRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.service.RegisterWithOIDC(c.Request.Context(), req.Email, req.Password, req.Code, req.DisplayName)
	if err != nil {
		h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondOIDCError(c, err)
		return
	}
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceOIDC)
	response.Success(c, gin.H{
		"user":       userpresenter.NewUserAuthBriefResp(res.User),
		"token":      res.Token,
		"expires_at": res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
