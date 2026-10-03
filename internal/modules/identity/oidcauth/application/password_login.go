package oidcauthapp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
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
	Cookies []storedCookie `json:"cookies"`
}

type storedCookie struct {
	Name  string `json:"n"`
	Value string `json:"v"`
}

func (s *Service) applicationName() string {
	cfg := s.currentConfig()
	if idx := strings.LastIndex(cfg.ApplicationID, "/"); idx >= 0 {
		return cfg.ApplicationID[idx+1:]
	}
	return cfg.ApplicationID
}

func newJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
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

func (s *Service) postJSON(ctx context.Context, client *http.Client, path string, payload map[string]interface{}) (*casdoorResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.currentConfig().Issuer+path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: http %d", ErrOIDCRemoteRejected, resp.StatusCode)
	}
	var out casdoorResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	return &out, nil
}

func (s *Service) getAccount(ctx context.Context, client *http.Client) (*IdentityVerified, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.currentConfig().Issuer+"/api/get-account", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: get-account http %d", ErrOIDCRemoteRejected, resp.StatusCode)
	}
	var out casdoorResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	if out.Status != "ok" {
		return nil, fmt.Errorf("%w: get-account %s", ErrOIDCRemoteRejected, out.Msg)
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
		return nil, fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	if strings.TrimSpace(account.ID) == "" || strings.TrimSpace(account.Email) == "" {
		return nil, fmt.Errorf("%w: account missing id/email", ErrOIDCRemoteRejected)
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
		return ErrOIDCInvalidCredentials
	case strings.Contains(lower, "too many times"):
		return ErrOIDCAccountFrozen
	case strings.Contains(lower, "turing test"):
		return ErrOIDCCaptchaRequired
	default:
		return fmt.Errorf("%w: %s", ErrOIDCRemoteRejected, msg)
	}
}

// LoginWithPassword 用邮箱/用户名+密码在 IdP 侧完成页内验证。
// 账号启用 MFA 时不报错,而是返回一次性挑战供 CompleteMFA 续接。
func (s *Service) LoginWithPassword(ctx context.Context, account, password string) (*IdentityVerified, *MFAChallenge, error) {
	if s == nil {
		return nil, nil, ErrOIDCAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled || cfg.ApplicationID == "" || cfg.Organization == "" {
		return nil, nil, ErrOIDCAuthConfigInvalid
	}
	account = strings.TrimSpace(account)
	password = strings.TrimSpace(password)
	if account == "" || password == "" {
		return nil, nil, ErrOIDCPayloadInvalid
	}

	client := &http.Client{Timeout: 10 * time.Second, Jar: newJar()}
	out, err := s.postJSON(ctx, client, "/api/login", map[string]interface{}{
		"type":         "login",
		"application":  s.applicationName(),
		"organization": cfg.Organization,
		"username":     account,
		"password":     password,
		"autoSignin":   true,
	})
	if err != nil {
		return nil, nil, err
	}
	switch {
	case out.Status == "ok" && out.Msg == "NextMfa":
		challenge, cerr := s.issueMFAChallenge(jarCookies(client.Jar, cfg.Issuer), out.Data)
		if cerr != nil {
			return nil, nil, cerr
		}
		return nil, challenge, nil
	case out.Status == "ok":
		verified, err := s.getAccount(ctx, client)
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
		return nil, ErrOIDCAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled || cfg.ApplicationID == "" || cfg.Organization == "" {
		return nil, ErrOIDCAuthConfigInvalid
	}
	challengeToken = strings.TrimSpace(challengeToken)
	mfaType = strings.TrimSpace(mfaType)
	passcode = strings.TrimSpace(passcode)
	if challengeToken == "" || mfaType == "" || passcode == "" {
		return nil, ErrOIDCPayloadInvalid
	}
	if s.mfaChallengeGet == nil {
		return nil, ErrOIDCAuthConfigInvalid
	}

	raw, ok, err := s.mfaChallengeGet(ctx, mfaChallengePrefix+challengeToken)
	if err != nil {
		return nil, err
	}
	if !ok || raw == "" {
		return nil, ErrOIDCMFAChallengeInvalid
	}
	var state mfaSessionState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, ErrOIDCMFAChallengeInvalid
	}

	client := &http.Client{Timeout: 10 * time.Second, Jar: restoreJar(state.Cookies, cfg.Issuer)}
	out, err := s.postJSON(ctx, client, "/api/login", map[string]interface{}{
		"type":         "login",
		"application":  s.applicationName(),
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
			return nil, ErrOIDCMFACodeInvalid
		}
		return nil, mapCasdoorLoginError(out.Msg)
	}
	verified, err := s.getAccount(ctx, client)
	if err != nil {
		return nil, err
	}
	_ = s.mfaChallengeDel(ctx, mfaChallengePrefix+challengeToken)
	return verified, nil
}

func (s *Service) issueMFAChallenge(cookies []storedCookie, propsRaw json.RawMessage) (*MFAChallenge, error) {
	if s.mfaChallengeSet == nil {
		return nil, ErrOIDCAuthConfigInvalid
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	token := base64URLNoPad(buf)
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
	state, err := json.Marshal(mfaSessionState{Cookies: cookies})
	if err != nil {
		return nil, err
	}
	ok, err := s.mfaChallengeSet(context.Background(), mfaChallengePrefix+token, string(state), mfaChallengeTokenTTL)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOIDCMFAChallengeInvalid
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
