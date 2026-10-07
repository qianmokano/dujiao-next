package ssoauthapp

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/config"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
	"github.com/dujiao-next/internal/shared/casdoorcaptcha"
)

// IdentityVerified is an identity returned by the configured Casdoor session API.
type IdentityVerified struct {
	Provider       string
	ProviderUserID string // Preserves the existing Casdoor ID / historical OIDC sub mapping.
	Email          string
	EmailVerified  bool
	Username       string
	DisplayName    string
	AvatarURL      string
	AvatarPresent  bool
	AuthAt         time.Time
}

type Option func(*Service)

// Service proxies Casdoor password, registration, MFA and CAPTCHA APIs.
type Service struct {
	cfgMu           sync.RWMutex
	cfg             config.SSOAuthConfig
	httpClient      *http.Client
	captchaStore    casdoorcaptcha.Store
	mfaChallengeSet MFAChallengeSetFunc
	mfaChallengeGet MFAChallengeGetFunc
	mfaChallengeDel MFAChallengeDelFunc
}

func NewService(cfg config.SSOAuthConfig, options ...Option) *Service {
	s := &Service{cfg: normalizeConfig(cfg), httpClient: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	for _, option := range options {
		if option != nil {
			option(s)
		}
	}
	return s
}

func WithCaptchaStore(store casdoorcaptcha.Store) Option {
	return func(s *Service) { s.captchaStore = store }
}

func (s *Service) SetConfig(cfg config.SSOAuthConfig) {
	if s == nil {
		return
	}
	s.cfgMu.Lock()
	s.cfg = normalizeConfig(cfg)
	s.cfgMu.Unlock()
}

func (s *Service) PublicConfig() map[string]interface{} {
	if s == nil {
		return map[string]interface{}{"enabled": false, "only_enabled": false, "display_name": ""}
	}
	cfg := s.currentConfig()
	parts := strings.Split(cfg.ApplicationID, "/")
	name := ""
	if len(parts) == 2 {
		name = parts[1]
	}
	return map[string]interface{}{
		"enabled":      cfg.Enabled && settingssecurity.SSOEndpointsReady(settingssecurity.DefaultSSOAuthSetting(cfg)),
		"only_enabled": cfg.OnlyEnabled, "display_name": cfg.DisplayName,
		"account_url":        settingssecurity.SSOIdentityPageURL(cfg.Issuer, "/account"),
		"password_reset_url": identityPageURL(cfg, "forget", name),
	}
}

// OnlyEnabled does not depend on provider availability, so faults cannot reopen local credentials.
func (s *Service) OnlyEnabled() bool { return s != nil && s.currentConfig().OnlyEnabled }

func (s *Service) currentConfig() config.SSOAuthConfig {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return normalizeConfig(s.cfg)
}

func (s *Service) captchaClient(cfg config.SSOAuthConfig) *casdoorcaptcha.Client {
	return casdoorcaptcha.New(casdoorcaptcha.Config{Issuer: cfg.Issuer, Application: cfg.ApplicationID, Organization: cfg.Organization}, s.captchaStore, s.httpClient)
}

func (s *Service) PrepareCaptcha(ctx context.Context, action, account string) (*casdoorcaptcha.Challenge, error) {
	if s == nil {
		return nil, ErrSSOAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled {
		return nil, ErrSSOAuthDisabled
	}
	return s.captchaClient(cfg).Prepare(ctx, action, account)
}

func identityPageURL(cfg config.SSOAuthConfig, page, name string) string {
	if name == "" {
		return ""
	}
	return settingssecurity.SSOIdentityPageURL(cfg.Issuer, "/"+page+"/"+url.PathEscape(name))
}

func optionalAvatar(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func normalizeConfig(cfg config.SSOAuthConfig) config.SSOAuthConfig {
	cfg.Issuer = strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	cfg.DisplayName = strings.TrimSpace(cfg.DisplayName)
	cfg.ApplicationID = strings.TrimSpace(cfg.ApplicationID)
	cfg.Organization = strings.TrimSpace(cfg.Organization)
	return cfg
}
