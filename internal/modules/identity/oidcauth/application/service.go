package oidcauthapp

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/config"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
)

// LoginIntentLogin / LoginIntentBind 标记一次授权流程的用途。
const (
	LoginIntentLogin = "login"
	LoginIntentBind  = "bind"
)

// IdentityVerified 是经过验签确认的通用 OIDC 身份。
type IdentityVerified struct {
	Provider       string
	ProviderUserID string // OIDC sub
	Email          string
	EmailVerified  bool
	Username       string
	DisplayName    string
	AvatarURL      string
	AuthAt         time.Time
}

type ReplaySetNXFunc func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
type OIDCStateSetFunc func(ctx context.Context, key string, value string, ttlSeconds int) (bool, error)
type OIDCStateTakeFunc func(ctx context.Context, key string) (string, bool, error)

type Option func(*Service)

func WithReplaySetNX(replaySetNX ReplaySetNXFunc) Option {
	return func(service *Service) {
		if replaySetNX != nil {
			service.replaySetNX = replaySetNX
		}
	}
}

func WithOIDCStateStore(set OIDCStateSetFunc, take OIDCStateTakeFunc) Option {
	return func(service *Service) {
		if set != nil && take != nil {
			service.oidcStateSet = set
			service.oidcStateTake = take
		}
	}
}

type oidcDiscoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// Service 实现面向任意 OIDC 提供方（本部署为 Casdoor）的授权码 + PKCE 客户端。
type Service struct {
	cfgMu       sync.RWMutex
	cfg         config.OIDCAuthConfig
	replaySetNX ReplaySetNXFunc

	httpClient    *http.Client
	oidcStateSet  OIDCStateSetFunc
	oidcStateTake OIDCStateTakeFunc

	mfaChallengeSet MFAChallengeSetFunc
	mfaChallengeGet MFAChallengeGetFunc
	mfaChallengeDel MFAChallengeDelFunc

	discoveryMu        sync.Mutex
	discovery          *oidcDiscoveryDocument
	discoveryFetchedAt time.Time

	jwksMu        sync.Mutex
	jwksCache     map[string]*rsa.PublicKey
	jwksFetchedAt time.Time
}

func NewService(cfg config.OIDCAuthConfig, options ...Option) *Service {
	svc := &Service{cfg: normalizeConfig(cfg)}
	svc.httpClient = &http.Client{Timeout: 10 * time.Second}
	svc.jwksCache = map[string]*rsa.PublicKey{}
	for _, option := range options {
		if option != nil {
			option(svc)
		}
	}
	return svc
}

// SetConfig 更新运行时配置，并使端点缓存失效。
func (s *Service) SetConfig(cfg config.OIDCAuthConfig) {
	if s == nil {
		return
	}
	s.cfgMu.Lock()
	s.cfg = normalizeConfig(cfg)
	s.cfgMu.Unlock()
	s.discoveryMu.Lock()
	s.discovery = nil
	s.discoveryFetchedAt = time.Time{}
	s.discoveryMu.Unlock()
	s.jwksMu.Lock()
	s.jwksCache = map[string]*rsa.PublicKey{}
	s.jwksFetchedAt = time.Time{}
	s.jwksMu.Unlock()
}

// PublicConfig 返回前台可见配置。
func (s *Service) PublicConfig() map[string]interface{} {
	if s == nil {
		return map[string]interface{}{"enabled": false, "display_name": ""}
	}
	cfg := s.currentConfig()
	enabled := cfg.Enabled && settingssecurity.OIDCEndpointsReady(settingssecurity.OIDCAuthSetting{
		Enabled: cfg.Enabled, Issuer: cfg.Issuer, ClientID: cfg.ClientID,
		ClientSecret: cfg.ClientSecret, RedirectURI: cfg.RedirectURI,
	})
	applicationName := cfg.ApplicationID
	if index := strings.LastIndex(applicationName, "/"); index >= 0 {
		applicationName = applicationName[index+1:]
	}
	return map[string]interface{}{
		"enabled":            enabled,
		"only_enabled":       cfg.OnlyEnabled,
		"display_name":       cfg.DisplayName,
		"account_url":        identityPageURL(cfg, "login", cfg.Organization),
		"password_reset_url": identityPageURL(cfg, "forget", applicationName),
	}
}

// OnlyEnabled reads the policy independently of provider availability so a
// disabled or unavailable IdP cannot reopen local authentication.
func (s *Service) OnlyEnabled() bool {
	return s != nil && s.currentConfig().OnlyEnabled
}

func (s *Service) currentConfig() config.OIDCAuthConfig {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return normalizeConfig(s.cfg)
}

func identityPageURL(cfg config.OIDCAuthConfig, page, name string) string {
	issuer, err := url.Parse(cfg.Issuer)
	if err != nil || (issuer.Scheme != "http" && issuer.Scheme != "https") || issuer.Host == "" || name == "" {
		return ""
	}
	return cfg.Issuer + "/" + page + "/" + url.PathEscape(name)
}

func normalizeConfig(cfg config.OIDCAuthConfig) config.OIDCAuthConfig {
	cfg.Issuer = strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.RedirectURI = strings.TrimRight(strings.TrimSpace(cfg.RedirectURI), "/")
	cfg.DisplayName = strings.TrimSpace(cfg.DisplayName)
	cfg.ApplicationID = strings.TrimSpace(cfg.ApplicationID)
	cfg.Organization = strings.TrimSpace(cfg.Organization)
	return cfg
}
