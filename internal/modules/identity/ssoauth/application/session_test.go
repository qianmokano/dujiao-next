package ssoauthapp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/shared/casdoorcaptcha"
)

type captchaMemory struct{ values map[string]string }

func (s *captchaMemory) Set(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	s.values[key] = value
	return true, nil
}
func (s *captchaMemory) Take(_ context.Context, key string) (string, bool, error) {
	value, ok := s.values[key]
	delete(s.values, key)
	return value, ok, nil
}

func TestCaptchaProofForwardingAndAtomicConsumption(t *testing.T) {
	for _, provider := range []string{"Default", "Cloudflare Turnstile"} {
		t.Run(provider, func(t *testing.T) {
			var submitted int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/get-captcha-status":
					if r.URL.Query().Get("application") != "store" || r.URL.Query().Get("organization") != "kano" {
						t.Error("unconfigured application queried")
					}
					_, _ = w.Write([]byte(`{"status":"ok","data":true}`))
				case "/api/get-captcha":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "data": map[string]string{"type": provider, "captchaId": "image-id", "captchaImage": "aGVsbG8=", "clientId": "public-key", "clientSecret": "never-expose"}})
				case "/api/login", "/api/signup", "/api/send-verification-code":
					submitted++
					var kind, answer, imageID string
					if r.URL.Path == "/api/send-verification-code" {
						_ = r.ParseForm()
						kind = r.Form.Get("captchaType")
						answer = r.Form.Get("captchaToken")
						imageID = r.Form.Get("clientSecret")
					} else {
						var body map[string]interface{}
						_ = json.NewDecoder(r.Body).Decode(&body)
						kind, _ = body["captchaType"].(string)
						answer, _ = body["captchaToken"].(string)
						imageID, _ = body["clientSecret"].(string)
						if body["password"] != " raw password " {
							t.Error("password was altered")
						}
					}
					if kind != provider || answer != "proof" || (provider == "Default" && imageID != "image-id") || (provider != "Default" && imageID != "") {
						t.Fatalf("incorrect CAPTCHA forwarding: %q %q %q", kind, answer, imageID)
					}
					http.SetCookie(w, &http.Cookie{Name: "casdoor_session_id", Value: "server-only"})
					_, _ = w.Write([]byte(`{"status":"ok","data":"kano/buyer"}`))
				case "/api/get-account":
					_, _ = w.Write([]byte(`{"status":"ok","data":{"id":"original-subject","name":"buyer","displayName":"Original","email":"buyer@example.com","emailVerified":true,"avatar":"https://auth.example/avatar.png"}}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			store := &captchaMemory{values: map[string]string{}}
			svc := NewService(config.SSOAuthConfig{Enabled: true, Issuer: server.URL, ApplicationID: "admin/store", Organization: "kano"}, nil, WithCaptchaStore(store))
			for _, action := range []string{casdoorcaptcha.ActionLogin, casdoorcaptcha.ActionSendCode, casdoorcaptcha.ActionRegister} {
				challenge, err := svc.PrepareCaptcha(context.Background(), action, "buyer@example.com")
				if err != nil || !challenge.Required {
					t.Fatalf("challenge=%v err=%v", challenge, err)
				}
				raw, _ := json.Marshal(challenge)
				if strings.Contains(string(raw), "never-expose") || strings.Contains(string(raw), "image-id") {
					t.Fatal("provider secret exposed")
				}
				proof := &casdoorcaptcha.Proof{Challenge: challenge.Token, Answer: "proof"}
				switch action {
				case casdoorcaptcha.ActionLogin:
					identity, mfa, err := svc.LoginWithPassword(context.Background(), "buyer@example.com", " raw password ", proof)
					if err != nil || mfa != nil || identity.ProviderUserID != "original-subject" || !identity.AvatarPresent {
						t.Fatalf("identity=%v mfa=%v err=%v", identity, mfa, err)
					}
				case casdoorcaptcha.ActionSendCode:
					if err := svc.SendRegisterCode(context.Background(), "buyer@example.com", proof); err != nil {
						t.Fatal(err)
					}
				case casdoorcaptcha.ActionRegister:
					if _, err := svc.RegisterWithPassword(context.Background(), "buyer@example.com", " raw password ", "code", "Buyer", proof); err != nil {
						t.Fatal(err)
					}
				}
				if _, _, err := svc.LoginWithPassword(context.Background(), "buyer@example.com", "password", proof); !errors.Is(err, casdoorcaptcha.ErrChallenge) {
					t.Fatalf("replay accepted: %v", err)
				}
			}
			if submitted != 3 {
				t.Fatalf("invalid requests reached provider: %d", submitted)
			}
		})
	}
}

func TestSessionValidationAndProviderFailures(t *testing.T) {
	ctx := context.Background()
	var nilService *Service
	nilService.SetConfig(config.SSOAuthConfig{})
	if _, _, err := nilService.LoginWithPassword(ctx, "buyer", "pw"); err == nil {
		t.Fatal("nil login accepted")
	}
	if err := nilService.SendRegisterCode(ctx, "buyer@example.com"); err == nil {
		t.Fatal("nil code accepted")
	}
	if _, err := nilService.RegisterWithPassword(ctx, "buyer@example.com", "pw", "code", ""); err == nil {
		t.Fatal("nil registration accepted")
	}
	if _, err := nilService.CompleteMFA(ctx, "challenge", "otp", "code"); err == nil {
		t.Fatal("nil MFA accepted")
	}
	if _, err := nilService.PrepareCaptcha(ctx, "login", "buyer"); err == nil {
		t.Fatal("nil CAPTCHA accepted")
	}
	disabled := NewService(config.SSOAuthConfig{}, WithMFAChallengeStore(nil, nil, nil))
	if _, err := disabled.PrepareCaptcha(ctx, "login", "buyer"); err != ErrSSOAuthDisabled {
		t.Fatal(err)
	}
	if _, _, err := disabled.LoginWithPassword(ctx, "buyer", "pw"); err == nil {
		t.Fatal("disabled login accepted")
	}
	if err := disabled.SendRegisterCode(ctx, "buyer@example.com"); err == nil {
		t.Fatal("disabled code accepted")
	}
	if _, err := disabled.RegisterWithPassword(ctx, "buyer@example.com", "pw", "code", ""); err == nil {
		t.Fatal("disabled registration accepted")
	}
	if _, err := disabled.CompleteMFA(ctx, "challenge", "otp", "code"); err == nil {
		t.Fatal("disabled MFA accepted")
	}
	mux := http.NewServeMux()
	responseStatus := 200
	responseBody := `invalid-json`
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseStatus)
		_, _ = w.Write([]byte(responseBody))
	})
	svc, store := newTestInpageService(t, mux)
	WithMFAChallengeStore(store.set, store.take, store.del)(svc)
	if _, _, err := svc.LoginWithPassword(ctx, "", "pw"); err != ErrSSOPayloadInvalid {
		t.Fatal(err)
	}
	if err := svc.SendRegisterCode(ctx, "invalid"); err != ErrSSOPayloadInvalid {
		t.Fatal(err)
	}
	if _, err := svc.RegisterWithPassword(ctx, "invalid", "pw", "code", ""); err != ErrSSOPayloadInvalid {
		t.Fatal(err)
	}
	if _, err := svc.CompleteMFA(ctx, "", "otp", "code"); err != ErrSSOPayloadInvalid {
		t.Fatal(err)
	}
	if _, _, err := svc.LoginWithPassword(ctx, "buyer", "pw"); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	if err := svc.SendRegisterCode(ctx, "buyer@example.com"); err == nil {
		t.Fatal("invalid code response accepted")
	}
	if _, err := svc.RegisterWithPassword(ctx, "buyer@example.com", "pw", "code", ""); err == nil {
		t.Fatal("invalid signup response accepted")
	}
	responseStatus = 503
	if _, _, err := svc.LoginWithPassword(ctx, "buyer", "pw"); err == nil {
		t.Fatal("upstream status accepted")
	}
	if err := svc.SendRegisterCode(ctx, "buyer@example.com"); err == nil {
		t.Fatal("upstream status accepted")
	}
	responseStatus = 200
	responseBody = `{"status":"error","msg":"Turing test failed"}`
	if _, _, err := svc.LoginWithPassword(ctx, "buyer", "pw"); err != ErrSSOCaptchaRequired {
		t.Fatal(err)
	}
	if err := svc.SendRegisterCode(ctx, "buyer@example.com"); err != ErrSSOCaptchaRequired {
		t.Fatal(err)
	}
	if _, err := svc.RegisterWithPassword(ctx, "buyer@example.com", "pw", "code", ""); err != ErrSSOCaptchaRequired {
		t.Fatal(err)
	}
	responseBody = `{"status":"ok","data":{}}`
	if _, _, err := svc.LoginWithPassword(ctx, "buyer", "pw"); err == nil {
		t.Fatal("missing verified identity accepted")
	}
	if applicationName(config.SSOAuthConfig{ApplicationID: "store"}) != "store" {
		t.Fatal("application parsing")
	}
	if optionalAvatar(nil) != "" {
		t.Fatal("missing avatar")
	}
	if len(synthesizeUsername(strings.Repeat("long", 20)+"@example.com", "suffix")) > 39 {
		t.Fatal("username length")
	}
}

func TestMFARejectsChangedApplicationAndBrokenState(t *testing.T) {
	svc, store := newTestInpageService(t, http.NewServeMux())
	ctx := context.Background()
	cfg := svc.currentConfig()
	challenge, err := svc.issueMFAChallenge(ctx, cfg, []storedCookie{{Name: "session", Value: "secret"}}, json.RawMessage(`invalid`))
	if err != nil || len(challenge.Props) != 1 {
		t.Fatalf("challenge=%v err=%v", challenge, err)
	}
	cfg.ApplicationID = "admin/changed"
	svc.SetConfig(cfg)
	if _, err := svc.CompleteMFA(ctx, challenge.Token, "otp", "123456"); err != ErrSSOMFAChallengeInvalid {
		t.Fatal("configuration change accepted", err)
	}
	store.kv[mfaChallengePrefix+challenge.Token] = "bad-json"
	if _, err := svc.CompleteMFA(ctx, challenge.Token, "otp", "123456"); err != ErrSSOMFAChallengeInvalid {
		t.Fatal("malformed state accepted", err)
	}
	svc.mfaChallengeGet = nil
	if _, err := svc.CompleteMFA(ctx, challenge.Token, "otp", "123456"); err == nil {
		t.Fatal("missing store accepted")
	}
	svc.mfaChallengeSet = nil
	if _, err := svc.issueMFAChallenge(ctx, cfg, nil, nil); err == nil {
		t.Fatal("missing store accepted")
	}
	svc.mfaChallengeSet = func(context.Context, string, string, int) (bool, error) { return false, errors.New("offline") }
	if _, err := svc.issueMFAChallenge(ctx, cfg, nil, nil); err == nil {
		t.Fatal("Redis failure ignored")
	}
	svc.mfaChallengeSet = func(context.Context, string, string, int) (bool, error) { return false, nil }
	if _, err := svc.issueMFAChallenge(ctx, cfg, nil, nil); err != ErrSSOMFAChallengeInvalid {
		t.Fatal("collision accepted")
	}
}

func TestAuthenticationRefusesRedirects(t *testing.T) {
	var leaked bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer destination.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	})
	svc, _ := newTestInpageService(t, mux)
	if _, _, err := svc.LoginWithPassword(context.Background(), "buyer", "password"); err == nil || leaked {
		t.Fatal("credentials forwarded through redirect")
	}
}
