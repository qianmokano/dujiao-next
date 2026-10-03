package ssoauthapp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/shared/casdoorcaptcha"
)

// MFAProp 是 Casdoor NextMfa 响应里的一种二步验证方式。
type MFAProp struct {
	MfaType string `json:"mfa_type"`
}

// MFAChallenge 是页内登录遇到 MFA 账号时下发的一次性挑战。
type MFAChallenge struct {
	Token string    `json:"token"`
	Props []MFAProp `json:"props"`
}

const (
	mfaChallengePrefix   = "oidc:mfa:challenge:"
	mfaChallengeTokenTTL = 600 // seconds
)

type MFAChallengeSetFunc func(ctx context.Context, key string, value string, ttlSeconds int) (bool, error)
type MFAChallengeGetFunc func(ctx context.Context, key string) (string, bool, error)
type MFAChallengeDelFunc func(ctx context.Context, key string) error

type mfaSessionState struct {
	Issuer       string         `json:"issuer"`
	Application  string         `json:"application"`
	Organization string         `json:"organization"`
	Cookies      []storedCookie `json:"cookies"`
}

type storedCookie struct {
	Name  string `json:"n"`
	Value string `json:"v"`
}

func applicationName(cfg config.SSOAuthConfig) string {
	if idx := strings.LastIndex(cfg.ApplicationID, "/"); idx >= 0 {
		return cfg.ApplicationID[idx+1:]
	}
	return cfg.ApplicationID
}

func newJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}

func (s *Service) sessionClient(jar http.CookieJar) *http.Client {
	client := *s.httpClient
	client.Jar = jar
	return &client
}

func jarCookies(jar http.CookieJar, issuer string) []storedCookie {
	u, _ := url.Parse(issuer)
	out := []storedCookie{}
	for _, c := range jar.Cookies(u) {
		out = append(out, storedCookie{Name: c.Name, Value: c.Value})
	}
	return out
}

func restoreJar(cookies []storedCookie, issuer string) http.CookieJar {
	jar := newJar()
	u, _ := url.Parse(issuer)
	httpCookies := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		httpCookies = append(httpCookies, &http.Cookie{Name: c.Name, Value: c.Value})
	}
	jar.SetCookies(u, httpCookies)
	return jar
}

type casdoorResponse struct {
	Status string          `json:"status"`
	Msg    string          `json:"msg"`
	Sub    string          `json:"sub"`
	Data   json.RawMessage `json:"data"`
}

func (s *Service) postJSON(ctx context.Context, client *http.Client, issuer, path string, payload map[string]interface{}) (*casdoorResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSORemoteRejected, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: http %d", ErrSSORemoteRejected, resp.StatusCode)
	}
	var out casdoorResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSORemoteRejected, err)
	}
	return &out, nil
}

func (s *Service) getAccount(ctx context.Context, client *http.Client, cfg config.SSOAuthConfig) (*IdentityVerified, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Issuer+"/api/get-account", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSORemoteRejected, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: get-account http %d", ErrSSORemoteRejected, resp.StatusCode)
	}
	var out casdoorResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSORemoteRejected, err)
	}
	if out.Status != "ok" {
		return nil, fmt.Errorf("%w: get-account %s", ErrSSORemoteRejected, out.Msg)
	}
	var account struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		DisplayName   string  `json:"displayName"`
		Email         string  `json:"email"`
		EmailVerified bool    `json:"emailVerified"`
		Avatar        *string `json:"avatar"`
	}
	if err := json.Unmarshal(out.Data, &account); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSORemoteRejected, err)
	}
	if strings.TrimSpace(account.ID) == "" || strings.TrimSpace(account.Email) == "" {
		return nil, fmt.Errorf("%w: account missing id/email", ErrSSORemoteRejected)
	}
	return &IdentityVerified{
		Provider:       constants.UserOAuthProviderOIDC,
		ProviderUserID: account.ID,
		Email:          strings.TrimSpace(account.Email),
		EmailVerified:  account.EmailVerified,
		Username:       strings.TrimSpace(account.Name),
		DisplayName:    strings.TrimSpace(account.DisplayName),
		AvatarURL:      optionalAvatar(account.Avatar),
		AvatarPresent:  account.Avatar != nil,
		AuthAt:         time.Now(),
	}, nil
}

// mapCasdoorLoginError 把 Casdoor 登录失败的 msg 归类为稳定错误。
func mapCasdoorLoginError(msg string) error {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "password or code is incorrect"), strings.Contains(lower, "user does not exist"):
		return ErrSSOInvalidCredentials
	case strings.Contains(lower, "too many times"):
		return ErrSSOAccountFrozen
	case strings.Contains(lower, "turing test"):
		return ErrSSOCaptchaRequired
	default:
		return fmt.Errorf("%w: %s", ErrSSORemoteRejected, msg)
	}
}

// LoginWithPassword 用邮箱/用户名+密码在 IdP 侧完成页内验证。
// 账号启用 MFA 时不报错,而是返回一次性挑战供 CompleteMFA 续接。
func (s *Service) LoginWithPassword(ctx context.Context, account, password string, proofs ...*casdoorcaptcha.Proof) (*IdentityVerified, *MFAChallenge, error) {
	if s == nil {
		return nil, nil, ErrSSOAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled || cfg.ApplicationID == "" || cfg.Organization == "" {
		return nil, nil, ErrSSOAuthConfigInvalid
	}
	account = strings.TrimSpace(account)
	if account == "" || strings.TrimSpace(password) == "" {
		return nil, nil, ErrSSOPayloadInvalid
	}

	var proof *casdoorcaptcha.Proof
	if len(proofs) > 0 {
		proof = proofs[0]
	}
	fields, err := s.captchaClient(cfg).Resolve(ctx, casdoorcaptcha.ActionLogin, account, proof)
	if err != nil {
		return nil, nil, err
	}
	client := s.sessionClient(newJar())
	payload := map[string]interface{}{
		"type": "login", "application": applicationName(cfg), "organization": cfg.Organization,
		"username": account, "password": password, "autoSignin": true,
	}
	fields.ApplyJSON(payload)
	out, err := s.postJSON(ctx, client, cfg.Issuer, "/api/login", payload)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case out.Status == "ok" && out.Msg == "NextMfa":
		challenge, cerr := s.issueMFAChallenge(ctx, cfg, jarCookies(client.Jar, cfg.Issuer), out.Data)
		if cerr != nil {
			return nil, nil, cerr
		}
		return nil, challenge, nil
	case out.Status == "ok":
		verified, err := s.getAccount(ctx, client, cfg)
		if err != nil {
			return nil, nil, err
		}
		return verified, nil, nil
	default:
		return nil, nil, mapCasdoorLoginError(out.Msg)
	}
}

// CompleteMFA 用挑战令牌与二步验证码续接登录。
func (s *Service) CompleteMFA(ctx context.Context, challengeToken, mfaType, passcode string) (*IdentityVerified, error) {
	if s == nil {
		return nil, ErrSSOAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled || cfg.ApplicationID == "" || cfg.Organization == "" {
		return nil, ErrSSOAuthConfigInvalid
	}
	challengeToken = strings.TrimSpace(challengeToken)
	mfaType = strings.TrimSpace(mfaType)
	passcode = strings.TrimSpace(passcode)
	if challengeToken == "" || mfaType == "" || passcode == "" {
		return nil, ErrSSOPayloadInvalid
	}
	if s.mfaChallengeGet == nil {
		return nil, ErrSSOAuthConfigInvalid
	}

	raw, ok, err := s.mfaChallengeGet(ctx, mfaChallengePrefix+challengeToken)
	if err != nil {
		return nil, err
	}
	if !ok || raw == "" {
		return nil, ErrSSOMFAChallengeInvalid
	}
	var state mfaSessionState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, ErrSSOMFAChallengeInvalid
	}

	if state.Issuer != cfg.Issuer || state.Application != cfg.ApplicationID || state.Organization != cfg.Organization {
		return nil, ErrSSOMFAChallengeInvalid
	}
	client := s.sessionClient(restoreJar(state.Cookies, cfg.Issuer))
	out, err := s.postJSON(ctx, client, cfg.Issuer, "/api/login", map[string]interface{}{
		"type":         "login",
		"application":  applicationName(cfg),
		"organization": cfg.Organization,
		"mfaType":      mfaType,
		"passcode":     passcode,
	})
	if err != nil {
		return nil, err
	}
	if out.Status != "ok" {
		lower := strings.ToLower(out.Msg)
		if strings.Contains(lower, "passcode") || strings.Contains(lower, "code") {
			return nil, ErrSSOMFACodeInvalid
		}
		return nil, mapCasdoorLoginError(out.Msg)
	}
	verified, err := s.getAccount(ctx, client, cfg)
	if err != nil {
		return nil, err
	}
	_ = s.mfaChallengeDel(ctx, mfaChallengePrefix+challengeToken)
	return verified, nil
}

func (s *Service) issueMFAChallenge(ctx context.Context, cfg config.SSOAuthConfig, cookies []storedCookie, propsRaw json.RawMessage) (*MFAChallenge, error) {
	if s.mfaChallengeSet == nil {
		return nil, ErrSSOAuthConfigInvalid
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	props := []MFAProp{}
	if len(propsRaw) > 0 {
		var raw []struct {
			MfaType string `json:"mfaType"`
		}
		if err := json.Unmarshal(propsRaw, &raw); err == nil {
			for _, p := range raw {
				if p.MfaType != "" {
					props = append(props, MFAProp{MfaType: p.MfaType})
				}
			}
		}
	}
	if len(props) == 0 {
		props = append(props, MFAProp{MfaType: "otp"})
	}
	state, err := json.Marshal(mfaSessionState{Cookies: cookies, Issuer: cfg.Issuer, Application: cfg.ApplicationID, Organization: cfg.Organization})
	if err != nil {
		return nil, err
	}
	ok, err := s.mfaChallengeSet(ctx, mfaChallengePrefix+token, string(state), mfaChallengeTokenTTL)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSSOMFAChallengeInvalid
	}
	return &MFAChallenge{Token: token, Props: props}, nil
}

// WithMFAChallengeStore 注入 MFA 挑战态存取实现(读不删,成功后删)。
func WithMFAChallengeStore(set MFAChallengeSetFunc, get MFAChallengeGetFunc, del MFAChallengeDelFunc) Option {
	return func(service *Service) {
		if set != nil && get != nil && del != nil {
			service.mfaChallengeSet = set
			service.mfaChallengeGet = get
			service.mfaChallengeDel = del
		}
	}
}
