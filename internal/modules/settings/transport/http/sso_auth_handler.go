package settingshttp

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/dujiao-next/internal/cache"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
	ginutil "github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
)

// SSOAuthAdminService 是后台通用 OIDC 登录设置端口。
type SSOAuthAdminService interface {
	GetSSOAuthSetting() (settingssecurity.SSOAuthSetting, error)
	PatchSSOAuthSetting(patch settingssecurity.SSOAuthSettingPatch) (settingssecurity.SSOAuthSetting, error)
	ApplyRuntime(setting settingssecurity.SSOAuthSetting)
}

// SSOAuthHandler 处理后台通用 OIDC 登录设置请求。
type SSOAuthHandler struct {
	ssoAuth SSOAuthAdminService
}

func NewSSOAuthHandler(ssoAuth SSOAuthAdminService) *SSOAuthHandler {
	if ssoAuth == nil {
		panic("settings oidc auth handler: ssoAuth is nil")
	}
	return &SSOAuthHandler{ssoAuth: ssoAuth}
}

// GetSSOAuth 获取通用 OIDC 登录配置。
func (h *SSOAuthHandler) GetSSOAuth(c *gin.Context) {
	setting, err := h.ssoAuth.GetSSOAuthSetting()
	if err != nil {
		ginutil.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, settingssecurity.MaskSSOAuthSettingForAdmin(setting))
}

// UpdateSSOAuth 更新通用 OIDC 登录配置并热更新登录服务。
func (h *SSOAuthHandler) UpdateSSOAuth(c *gin.Context) {
	var req settingssecurity.SSOAuthSettingPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}

	setting, err := h.ssoAuth.PatchSSOAuthSetting(req)
	if err != nil {
		switch {
		case errors.Is(err, settingssecurity.ErrSSOAuthConfigInvalid):
			ginutil.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
		default:
			ginutil.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		}
		return
	}

	h.ssoAuth.ApplyRuntime(setting)
	_ = cache.DelAllPublicConfig(c.Request.Context())
	response.Success(c, settingssecurity.MaskSSOAuthSettingForAdmin(setting))
}
