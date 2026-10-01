package settingsbootstrap

import (
	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/config"
	oidcauthapp "github.com/dujiao-next/internal/modules/identity/oidcauth/application"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
	settingstransport "github.com/dujiao-next/internal/modules/settings/transport/http"
)

func NewSMTPHandler(c *container.Container, cfg *config.Config) *settingstransport.SMTPHandler {
	return settingstransport.NewSMTPHandler(settingsSMTPAdapter{
		settings: c.SettingService, cfg: cfg, email: c.EmailSender,
	})
}

func NewCaptchaHandler(c *container.Container, cfg *config.Config) *settingstransport.CaptchaHandler {
	return settingstransport.NewCaptchaHandler(settingsCaptchaAdapter{
		settings: c.SettingService, cfg: cfg, captcha: c.CaptchaService,
	})
}

func NewTelegramAuthHandler(c *container.Container, cfg *config.Config) *settingstransport.TelegramAuthHandler {
	return settingstransport.NewTelegramAuthHandler(settingsTelegramAuthAdapter{
		settings: c.SettingService, cfg: cfg, telegramAuth: c.TelegramAuthService,
	})
}

func NewGoogleAuthHandler(c *container.Container, cfg *config.Config) *settingstransport.GoogleAuthHandler {
	return settingstransport.NewGoogleAuthHandler(settingsGoogleAuthAdapter{
		settings: c.SettingService, cfg: cfg, googleAuth: c.GoogleAuthService,
	})
}

func NewOIDCAuthHandler(c *container.Container, cfg *config.Config) *settingstransport.OIDCAuthHandler {
	return settingstransport.NewOIDCAuthHandler(settingsOIDCAuthAdapter{
		settings: c.SettingService, cfg: cfg, oidcAuth: c.OIDCAuthService,
	})
}

type settingsOIDCAuthAdapter struct {
	settings *settingsapp.Service
	cfg      *config.Config
	oidcAuth *oidcauthapp.Service
}

func (a settingsOIDCAuthAdapter) GetOIDCAuthSetting() (settingssecurity.OIDCAuthSetting, error) {
	return a.settings.GetOIDCAuthSetting(a.cfg.OIDCAuth)
}

func (a settingsOIDCAuthAdapter) PatchOIDCAuthSetting(patch settingssecurity.OIDCAuthSettingPatch) (settingssecurity.OIDCAuthSetting, error) {
	return a.settings.PatchOIDCAuthSetting(a.cfg.OIDCAuth, patch)
}

func (a settingsOIDCAuthAdapter) ApplyRuntime(setting settingssecurity.OIDCAuthSetting) {
	a.cfg.OIDCAuth = settingssecurity.OIDCAuthSettingToConfig(setting)
	if a.oidcAuth != nil {
		a.oidcAuth.SetConfig(a.cfg.OIDCAuth)
	}
}
