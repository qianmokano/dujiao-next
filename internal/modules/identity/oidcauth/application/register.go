package oidcauthapp

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encoding/json"
)

// SendRegisterCode 让 IdP 向邮箱发送注册验证码(method=signup)。
func (s *Service) SendRegisterCode(ctx context.Context, email string) error {
	if s == nil {
		return ErrOIDCAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled || cfg.ApplicationID == "" || cfg.Organization == "" {
		return ErrOIDCAuthConfigInvalid
	}
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return ErrOIDCPayloadInvalid
	}

	form := url.Values{}
	form.Set("dest", email)
	form.Set("type", "email")
	form.Set("applicationId", cfg.ApplicationID)
	form.Set("method", "signup")
	form.Set("captchaType", "none")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Issuer+"/api/send-verification-code", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%w: http %d", ErrOIDCRemoteRejected, resp.StatusCode)
	}
	var out casdoorResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("%w: %v", ErrOIDCRemoteRejected, err)
	}
	if out.Status != "ok" {
		lower := strings.ToLower(out.Msg)
		switch {
		case strings.Contains(lower, "already exists"):
			return ErrOIDCEmailExists
		case strings.Contains(lower, "only send one code"):
			return ErrOIDCCodeResendWait
		default:
			return fmt.Errorf("%w: %s", ErrOIDCRemoteRejected, out.Msg)
		}
	}
	return nil
}

// RegisterWithPassword 在 IdP 完成邮箱验证码注册并直接返回已验证身份。
// username 由邮箱本地部分合成,冲突时追加随机后缀重试一次。
func (s *Service) RegisterWithPassword(ctx context.Context, email, password, code, displayName string) (*IdentityVerified, error) {
	if s == nil {
		return nil, ErrOIDCAuthConfigInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled || cfg.ApplicationID == "" || cfg.Organization == "" {
		return nil, ErrOIDCAuthConfigInvalid
	}
	email = strings.TrimSpace(email)
	code = strings.TrimSpace(code)
	if email == "" || !strings.Contains(email, "@") || strings.TrimSpace(password) == "" || code == "" {
		return nil, ErrOIDCPayloadInvalid
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		if idx := strings.IndexByte(email, '@'); idx > 0 {
			displayName = email[:idx]
		} else {
			displayName = email
		}
	}

	username := synthesizeUsername(email, "")
	client := &http.Client{Timeout: 10 * time.Second, Jar: newJar()}
	for attempt := 0; attempt < 2; attempt++ {
		out, err := s.postJSON(ctx, client, "/api/signup", map[string]interface{}{
			"application":  s.applicationName(),
			"organization": cfg.Organization,
			"username":     username,
			"name":         displayName,
			"password":     password,
			"email":        email,
			"emailCode":    code,
			"autoSignin":   true,
		})
		if err != nil {
			return nil, err
		}
		if out.Status == "ok" {
			// 注册成功即会话有效,直接取账号,避免再一次登录计数
			return s.getAccount(ctx, client)
		}
		lower := strings.ToLower(out.Msg)
		switch {
		case strings.Contains(lower, "email"), strings.Contains(lower, "already exists"):
			// 邮箱层面的冲突直接透出;仅用户名冲突才值得重试
			if !strings.Contains(lower, "username") {
				if strings.Contains(lower, "code") {
					return nil, ErrOIDCCodeInvalid
				}
				return nil, ErrOIDCEmailExists
			}
		case strings.Contains(lower, "code"):
			return nil, ErrOIDCCodeInvalid
		default:
			return nil, fmt.Errorf("%w: %s", ErrOIDCRemoteRejected, out.Msg)
		}
		username = synthesizeUsername(email, randomSuffix(4))
	}
	return nil, ErrOIDCEmailExists
}

// synthesizeUsername 由邮箱本地部分生成合法 Casdoor 用户名:
// 仅 [a-z0-9_],不以数字开头,长度 >= 2,suffix 非空时追加。
func synthesizeUsername(email, suffix string) string {
	local := email
	if idx := strings.IndexByte(email, '@'); idx > 0 {
		local = email[:idx]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '+':
			b.WriteRune('_')
		}
	}
	name := b.String()
	name = strings.Trim(name, "_")
	if name == "" {
		name = "user"
	}
	if name[0] >= '0' && name[0] <= '9' || len(name) < 2 {
		name = "u" + name
	}
	name += suffix
	if len(name) > 39 {
		name = name[:39]
	}
	return name
}

func randomSuffix(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
