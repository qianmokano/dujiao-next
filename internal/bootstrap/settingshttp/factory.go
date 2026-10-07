package settingsbootstrap

import (
	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/config"
	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"
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

func NewSSOAuthHandler(c *container.Container, cfg *config.Config) *settingstransport.SSOAuthHandler {
	return settingstransport.NewSSOAuthHandler(settingsSSOAuthAdapter{
		settings: c.SettingService, cfg: cfg, ssoAuth: c.SSOAuthService,
	})
}

type settingsSSOAuthAdapter struct {
	settings *settingsapp.Service
	cfg      *config.Config
	ssoAuth  *ssoauthapp.Service
}

func (a settingsSSOAuthAdapter) GetSSOAuthSetting() (settingssecurity.SSOAuthSetting, error) {
	return a.settings.GetSSOAuthSetting(a.cfg.SSOAuth)
}

func (a settingsSSOAuthAdapter) PatchSSOAuthSetting(patch settingssecurity.SSOAuthSettingPatch) (settingssecurity.SSOAuthSetting, error) {
	return a.settings.PatchSSOAuthSetting(a.cfg.SSOAuth, patch)
}

func (a settingsSSOAuthAdapter) ApplyRuntime(setting settingssecurity.SSOAuthSetting) {
	a.cfg.SSOAuth = settingssecurity.SSOAuthSettingToConfig(setting)
	if a.ssoAuth != nil {
		a.ssoAuth.SetConfig(a.cfg.SSOAuth)
	}
}
