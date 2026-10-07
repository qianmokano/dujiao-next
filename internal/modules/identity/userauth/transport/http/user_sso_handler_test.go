package userauthhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	captchacontract "github.com/dujiao-next/internal/modules/captcha/contract"
	captchahttp "github.com/dujiao-next/internal/modules/captcha/transport/http"
	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	settingshttp "github.com/dujiao-next/internal/modules/settings/transport/http"
	"github.com/dujiao-next/internal/shared/casdoorcaptcha"
	"github.com/gin-gonic/gin"
)

type ssoHTTPService struct {
	err   error
	mfa   bool
	calls int
	proof *casdoorcaptcha.Proof
}

func (s *ssoHTTPService) result() *AuthLoginResult {
	return &AuthLoginResult{User: &userdomain.User{ID: 42, Email: "buyer@example.com"}, Token: "business-session", ExpiresAt: time.Now().Add(time.Hour)}
}
func (s *ssoHTTPService) LoginWithSSOPassword(_ context.Context, _, _ string, proof *casdoorcaptcha.Proof) (*AuthLoginResult, *SSOMFAChallengeView, error) {
	s.calls++
	s.proof = proof
	if s.err != nil {
		return nil, nil, s.err
	}
	if s.mfa {
		return nil, &SSOMFAChallengeView{Token: "challenge", Props: []SSOMFAPropView{{MfaType: "otp"}}}, nil
	}
	return s.result(), nil, nil
}
func (s *ssoHTTPService) PrepareSSOCaptcha(_ context.Context, _, _ string) (*casdoorcaptcha.Challenge, error) {
	s.calls++
	return &casdoorcaptcha.Challenge{Required: false}, s.err
}
func (s *ssoHTTPService) CompleteSSOMFA(_ context.Context, _, _, _ string) (*AuthLoginResult, error) {
	s.calls++
	return s.result(), s.err
}
func (s *ssoHTTPService) SendSSORegisterCode(_ context.Context, _ string, proof *casdoorcaptcha.Proof) error {
	s.calls++
	s.proof = proof
	return s.err
}
func (s *ssoHTTPService) RegisterWithSSO(_ context.Context, _, _, _, _ string, proof *casdoorcaptcha.Proof) (*AuthLoginResult, error) {
	s.calls++
	s.proof = proof
	return s.result(), s.err
}

type ssoSiteCaptcha struct {
	err   error
	calls int
}

func (s *ssoSiteCaptcha) Verify(_ string, payload captchahttp.CaptchaPayloadRequest, _ string) error {
	s.calls++
	if payload.CaptchaCode != "site-code" {
		return captchacontract.ErrRequired
	}
	return s.err
}

func ssoHTTPRouter(service *ssoHTTPService, captcha CaptchaVerifier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	rate := func(c *gin.Context) { c.Next() }
	RegisterUserSSOAuthRoutes(router.Group("/api/v1/auth"), NewUserSSOHandler(service, nil, captcha), rate, rate)
	settingshttp.RegisterAdminSSOAuthRoutes(router.Group("/api/v1/admin"), &settingshttp.SSOAuthHandler{})
	return router
}
func ssoHTTPPost(router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func ssoHTTPSuccess(w *httptest.ResponseRecorder) bool {
	var body struct {
		StatusCode int `json:"status_code"`
	}
	return w.Code == http.StatusOK && json.Unmarshal(w.Body.Bytes(), &body) == nil && body.StatusCode == 0
}

func TestDeletedOIDCRoutesReturnNotFound(t *testing.T) {
	router := ssoHTTPRouter(&ssoHTTPService{}, nil)
	for _, endpoint := range []string{"/api/v1/auth/oidc/start", "/api/v1/auth/oidc/callback", "/api/v1/auth/oidc/password-login", "/api/v1/auth/oidc/mfa", "/api/v1/auth/oidc/register/send-code", "/api/v1/auth/oidc/register", "/api/v1/me/oidc", "/api/v1/me/oidc/start", "/api/v1/me/oidc/callback", "/api/v1/me/oidc/unbind", "/api/v1/admin/settings/oidc-auth"} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(method, endpoint, nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("%s %s returned %d", method, endpoint, w.Code)
			}
		}
	}
}

func TestSSOHTTPContractsAndSiteCaptcha(t *testing.T) {
	service := &ssoHTTPService{}
	site := &ssoSiteCaptcha{}
	router := ssoHTTPRouter(service, site)
	proof := `"captcha":{"challenge":"proof-id","answer":"provider-code"},"captcha_payload":{"captcha_code":"site-code"}`
	for _, test := range []struct{ endpoint, body string }{
		{"password-login", `{"email":"buyer@example.com","password":"pw",` + proof + `}`},
		{"register/send-code", `{"email":"buyer@example.com",` + proof + `}`},
		{"register", `{"email":"buyer@example.com","password":"pw","code":"email-code",` + proof + `}`},
		{"captcha", `{"action":"login","account":"buyer@example.com"}`},
		{"mfa", `{"challenge":"mfa","mfa_type":"otp","passcode":"123456"}`},
	} {
		w := ssoHTTPPost(router, "/api/v1/auth/sso/"+test.endpoint, test.body)
		if !ssoHTTPSuccess(w) {
			t.Fatalf("%s: %d %s", test.endpoint, w.Code, w.Body.String())
		}
		if test.endpoint != "captcha" && test.endpoint != "mfa" && (service.proof == nil || service.proof.Answer != "provider-code") {
			t.Fatal("provider proof lost")
		}
		before := service.calls
		w = ssoHTTPPost(router, "/api/v1/auth/sso/"+test.endpoint, `{}`)
		if ssoHTTPSuccess(w) || before != service.calls {
			t.Fatalf("invalid %s reached authentication", test.endpoint)
		}
	}
	if site.calls != 2 {
		t.Fatalf("site CAPTCHA checks=%d", site.calls)
	}
	before := service.calls
	w := ssoHTTPPost(router, "/api/v1/auth/sso/password-login", `{"email":"buyer@example.com","password":"pw"}`)
	if ssoHTTPSuccess(w) || before != service.calls {
		t.Fatal("missing site CAPTCHA reached IdP")
	}
}

func TestSSOMFAPendingCannotExposeSession(t *testing.T) {
	service := &ssoHTTPService{mfa: true}
	router := ssoHTTPRouter(service, nil)
	w := ssoHTTPPost(router, "/api/v1/auth/sso/password-login", `{"email":"buyer@example.com","password":"pw"}`)
	var envelope struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data["requires_mfa"] != true || envelope.Data["mfa_challenge"] == nil {
		t.Fatal(w.Body.String())
	}
	for _, key := range []string{"token", "user", "expires_at"} {
		if _, exists := envelope.Data[key]; exists {
			t.Fatalf("MFA pending exposed %s", key)
		}
	}
	service.err = ssoauthapp.ErrSSOMFACodeInvalid
	w = ssoHTTPPost(router, "/api/v1/auth/sso/mfa", `{"challenge":"mfa","mfa_type":"otp","passcode":"wrong"}`)
	if strings.Contains(w.Body.String(), "business-session") || ssoHTTPSuccess(w) {
		t.Fatal("wrong MFA issued session")
	}
}

func TestSSOErrorsDoNotIssueSessions(t *testing.T) {
	for _, err := range []error{ssoauthapp.ErrSSOAuthDisabled, ssoauthapp.ErrSSOAuthConfigInvalid, ssoauthapp.ErrSSOPayloadInvalid, ErrUserOAuthIdentityExists, ErrUserOAuthAlreadyBound, ssoauthapp.ErrSSOInvalidCredentials, ssoauthapp.ErrSSOAccountFrozen, casdoorcaptcha.ErrUnsupported, casdoorcaptcha.ErrUnavailable, casdoorcaptcha.ErrChallenge, casdoorcaptcha.ErrInvalid, ssoauthapp.ErrSSOCaptchaRequired, ssoauthapp.ErrSSOMFAChallengeInvalid, ssoauthapp.ErrSSOMFACodeInvalid, ssoauthapp.ErrSSOCodeInvalid, ssoauthapp.ErrSSOEmailExists, ssoauthapp.ErrSSOCodeResendWait, ssoauthapp.ErrSSORemoteRejected, ErrUserDisabled, ErrEmailNotVerified, ErrRegistrationDisabled, errors.New("private-secret")} {
		service := &ssoHTTPService{err: err}
		router := ssoHTTPRouter(service, nil)
		for _, endpoint := range []string{"password-login", "mfa", "register/send-code", "register", "captcha"} {
			w := ssoHTTPPost(router, "/api/v1/auth/sso/"+endpoint, `{"email":"buyer@example.com","password":"pw","code":"code","challenge":"challenge","mfa_type":"otp","passcode":"123456","action":"login","account":"buyer"}`)
			if ssoHTTPSuccess(w) || strings.Contains(w.Body.String(), "business-session") || strings.Contains(w.Body.String(), "private-secret") {
				t.Fatalf("%s: %s", endpoint, w.Body.String())
			}
		}
	}
}
