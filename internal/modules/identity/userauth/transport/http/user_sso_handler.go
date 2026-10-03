package userauthhttp

import (
	"context"
	"errors"

	"github.com/dujiao-next/internal/constants"
	captchahttp "github.com/dujiao-next/internal/modules/captcha/transport/http"
	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"
	userpresenter "github.com/dujiao-next/internal/modules/identity/userauth/transport/presenter"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/dujiao-next/internal/shared/casdoorcaptcha"

	"github.com/gin-gonic/gin"
)

// UserSSOService 是页内通行证端点所需的最小端口。
type UserSSOService interface {
	LoginWithSSOPassword(ctx context.Context, account, password string, captcha *casdoorcaptcha.Proof) (*AuthLoginResult, *SSOMFAChallengeView, error)
	PrepareSSOCaptcha(ctx context.Context, action, account string) (*casdoorcaptcha.Challenge, error)
	CompleteSSOMFA(ctx context.Context, challenge, mfaType, passcode string) (*AuthLoginResult, error)
	SendSSORegisterCode(ctx context.Context, email string, captcha *casdoorcaptcha.Proof) error
	RegisterWithSSO(ctx context.Context, email, password, code, displayName string, captcha *casdoorcaptcha.Proof) (*AuthLoginResult, error)
}

// SSOMFAChallengeView 是页内登录的 MFA 挑战视图。
type SSOMFAChallengeView struct {
	Token string           `json:"token"`
	Props []SSOMFAPropView `json:"props"`
}

// SSOMFAPropView 是一种可选的二步验证方式。
type SSOMFAPropView struct {
	MfaType string `json:"mfa_type"`
}

// UserSSOHandler 处理页内通行证认证请求。
type UserSSOHandler struct {
	service  UserSSOService
	recorder LoginRecorder
	captcha  CaptchaVerifier
}

func NewUserSSOHandler(service UserSSOService, recorder LoginRecorder, captcha CaptchaVerifier) *UserSSOHandler {
	if service == nil {
		panic("user sso handler: service is nil")
	}
	return &UserSSOHandler{service: service, recorder: recorder, captcha: captcha}
}

func respondSSOError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ssoauthapp.ErrSSOAuthDisabled):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_auth_disabled", nil)
	case errors.Is(err, ssoauthapp.ErrSSOAuthConfigInvalid):
		ginutil.RespondError(c, response.CodeInternal, "error.sso_auth_config_invalid", err)
	case errors.Is(err, ssoauthapp.ErrSSOPayloadInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_payload_invalid", nil)
	case errors.Is(err, ErrUserOAuthIdentityExists):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_bind_conflict", nil)
	case errors.Is(err, ErrUserOAuthAlreadyBound):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_already_bound", nil)
	case errors.Is(err, ssoauthapp.ErrSSOInvalidCredentials):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_invalid_credentials", nil)
	case errors.Is(err, ssoauthapp.ErrSSOAccountFrozen):
		ginutil.RespondError(c, response.CodeTooManyRequests, "error.sso_account_frozen", nil)
	case errors.Is(err, casdoorcaptcha.ErrUnsupported):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_captcha_unsupported", nil)
	case errors.Is(err, casdoorcaptcha.ErrUnavailable):
		ginutil.RespondError(c, response.CodeInternal, "error.sso_captcha_unavailable", nil)
	case errors.Is(err, casdoorcaptcha.ErrChallenge), errors.Is(err, casdoorcaptcha.ErrInvalid), errors.Is(err, ssoauthapp.ErrSSOCaptchaRequired):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_captcha_required", nil)
	case errors.Is(err, ssoauthapp.ErrSSOMFAChallengeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_mfa_challenge_invalid", nil)
	case errors.Is(err, ssoauthapp.ErrSSOMFACodeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_mfa_code_invalid", nil)
	case errors.Is(err, ssoauthapp.ErrSSOCodeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_code_invalid", nil)
	case errors.Is(err, ssoauthapp.ErrSSOEmailExists):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_email_exists", nil)
	case errors.Is(err, ssoauthapp.ErrSSOCodeResendWait):
		ginutil.RespondError(c, response.CodeBadRequest, "error.sso_code_resend_wait", nil)
	case errors.Is(err, ssoauthapp.ErrSSORemoteRejected):
		ginutil.RespondError(c, response.CodeInternal, "error.sso_remote_rejected", err)
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

func (h *UserSSOHandler) recordLogin(c *gin.Context, email string, userID uint, status, failReason, source string) {
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

type ssoPasswordLoginRequest struct {
	Email          string                            `json:"email" binding:"required"`
	Password       string                            `json:"password" binding:"required"`
	Captcha        *casdoorcaptcha.Proof             `json:"captcha"`
	CaptchaPayload captchahttp.CaptchaPayloadRequest `json:"captcha_payload"`
}

func (h *UserSSOHandler) SSOCaptcha(c *gin.Context) {
	var req struct {
		Action  string `json:"action" binding:"required"`
		Account string `json:"account" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	challenge, err := h.service.PrepareSSOCaptcha(c.Request.Context(), req.Action, req.Account)
	if err != nil {
		respondSSOError(c, err)
		return
	}
	response.Success(c, challenge)
}

// SSOPasswordLogin 页内直连登录(凭据由后端代理交 IdP 校验)。
func (h *UserSSOHandler) SSOPasswordLogin(c *gin.Context) {
	var req ssoPasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonBadRequest, constants.LoginLogSourceOIDC)
		ginutil.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	if h.captcha != nil {
		if err := h.captcha.Verify(constants.CaptchaSceneLogin, req.CaptchaPayload, c.ClientIP()); err != nil {
			respondCaptchaError(c, err)
			return
		}
	}
	res, challenge, err := h.service.LoginWithSSOPassword(c.Request.Context(), req.Email, req.Password, req.Captcha)
	if err != nil {
		h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondSSOError(c, err)
		return
	}
	if challenge != nil {
		props := make([]SSOMFAPropView, 0, len(challenge.Props))
		for _, p := range challenge.Props {
			props = append(props, SSOMFAPropView{MfaType: p.MfaType})
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

type ssoMFARequest struct {
	Challenge string `json:"challenge" binding:"required"`
	MfaType   string `json:"mfa_type" binding:"required"`
	Passcode  string `json:"passcode" binding:"required"`
}

// SSOMFAComplete 续接页内登录的 MFA 二步验证。
func (h *UserSSOHandler) SSOMFAComplete(c *gin.Context) {
	var req ssoMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.service.CompleteSSOMFA(c.Request.Context(), req.Challenge, req.MfaType, req.Passcode)
	if err != nil {
		h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondSSOError(c, err)
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

type ssoRegisterCodeRequest struct {
	Email          string                            `json:"email" binding:"required"`
	Captcha        *casdoorcaptcha.Proof             `json:"captcha"`
	CaptchaPayload captchahttp.CaptchaPayloadRequest `json:"captcha_payload"`
}

// SSORegisterSendCode 页内注册:发送邮箱验证码(经 IdP)。
func (h *UserSSOHandler) SSORegisterSendCode(c *gin.Context) {
	var req ssoRegisterCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	if h.captcha != nil {
		if err := h.captcha.Verify(constants.CaptchaSceneRegisterSendCode, req.CaptchaPayload, c.ClientIP()); err != nil {
			respondCaptchaError(c, err)
			return
		}
	}
	if err := h.service.SendSSORegisterCode(c.Request.Context(), req.Email, req.Captcha); err != nil {
		respondSSOError(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}

type ssoRegisterRequest struct {
	Email          string                            `json:"email" binding:"required"`
	Password       string                            `json:"password" binding:"required"`
	Code           string                            `json:"code" binding:"required"`
	DisplayName    string                            `json:"display_name"`
	Captcha        *casdoorcaptcha.Proof             `json:"captcha"`
	CaptchaPayload captchahttp.CaptchaPayloadRequest `json:"captcha_payload"`
}

// SSORegister 页内注册:IdP 完成邮箱验证码开户并即时登录。
func (h *UserSSOHandler) SSORegister(c *gin.Context) {
	var req ssoRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.service.RegisterWithSSO(c.Request.Context(), req.Email, req.Password, req.Code, req.DisplayName, req.Captcha)
	if err != nil {
		h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonOIDCInvalid, constants.LoginLogSourceOIDC)
		respondSSOError(c, err)
		return
	}
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceOIDC)
	response.Success(c, gin.H{
		"user":       userpresenter.NewUserAuthBriefResp(res.User),
		"token":      res.Token,
		"expires_at": res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
