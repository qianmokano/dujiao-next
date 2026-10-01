package oidcauthapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
)

func newTestInpageService(t *testing.T, mux *http.ServeMux) (*Service, *mfaStoreFake) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	svc := NewService(config.OIDCAuthConfig{
		Enabled:       true,
		Issuer:        srv.URL,
		ClientID:      "c",
		ClientSecret:  "s",
		RedirectURI:   "https://shop.example.com/auth/oidc/callback",
		ApplicationID: "admin/dujiao-store",
		Organization:  "kano",
	})
	store := &mfaStoreFake{kv: map[string]string{}}
	svc.mfaChallengeSet = store.set
	svc.mfaChallengeGet = store.take
	svc.mfaChallengeDel = store.del
	return svc, store
}

type mfaStoreFake struct {
	kv map[string]string
}

func (f *mfaStoreFake) set(ctx context.Context, key, value string, ttl int) (bool, error) {
	if _, ok := f.kv[key]; ok {
		return false, nil
	}
	f.kv[key] = value
	return true, nil
}

func (f *mfaStoreFake) take(ctx context.Context, key string) (string, bool, error) {
	v, ok := f.kv[key]
	return v, ok, nil
}

func (f *mfaStoreFake) del(ctx context.Context, key string) error {
	delete(f.kv, key)
	return nil
}

func TestLoginWithPasswordHappyPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "buyer@example.com" || body["password"] != "right" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"status":"error","msg":"password or code is incorrect, you have 4 remaining chances"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "casdoor_session_id", Value: "sess-1"})
		_, _ = w.Write([]byte(`{"status":"ok","msg":"","data":"kano/buyer"}`))
	})
	mux.HandleFunc("/api/get-account", func(w http.ResponseWriter, r *http.Request) {
		ck, err := r.Cookie("casdoor_session_id")
		if err != nil || ck.Value != "sess-1" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"Please login first"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","sub":"uuid-1","data":{"id":"uuid-1","name":"buyer","displayName":"Buyer","email":"buyer@example.com","emailVerified":true}}`))
	})
	svc, _ := newTestInpageService(t, mux)

	verified, challenge, err := svc.LoginWithPassword(context.Background(), "buyer@example.com", "right")
	if err != nil {
		t.Fatalf("LoginWithPassword: %v", err)
	}
	if challenge != nil {
		t.Fatalf("unexpected MFA challenge")
	}
	if verified.ProviderUserID != "uuid-1" || verified.Email != "buyer@example.com" || !verified.EmailVerified {
		t.Fatalf("verified = %+v", verified)
	}
	if verified.Provider != constants.UserOAuthProviderOIDC {
		t.Fatalf("provider = %q", verified.Provider)
	}

	if _, _, err := svc.LoginWithPassword(context.Background(), "buyer@example.com", "wrong"); err != ErrOIDCInvalidCredentials {
		t.Fatalf("wrong password err = %v, want ErrOIDCInvalidCredentials", err)
	}
}

func TestLoginWithPasswordFrozen(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","msg":"You have entered the wrong password or code too many times, please wait for 12 minutes to try again"}`))
	})
	svc, _ := newTestInpageService(t, mux)
	if _, _, err := svc.LoginWithPassword(context.Background(), "a@b.com", "x"); err != ErrOIDCAccountFrozen {
		t.Fatalf("err = %v, want ErrOIDCAccountFrozen", err)
	}
}

func TestLoginWithPasswordMFAFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["passcode"] == nil {
			http.SetCookie(w, &http.Cookie{Name: "casdoor_session_id", Value: "mfa-sess-1"})
			_, _ = w.Write([]byte(`{"status":"ok","msg":"NextMfa","data":[{"mfaType":"otp"},{"mfaType":"email"}]}`))
			return
		}
		if body["mfaType"] != "otp" || body["passcode"] != "123456" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"passcode is incorrect"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "casdoor_session_id", Value: "sess-2"})
		_, _ = w.Write([]byte(`{"status":"ok","msg":"","data":"kano/u"}`))
	})
	mux.HandleFunc("/api/get-account", func(w http.ResponseWriter, r *http.Request) {
		ck, _ := r.Cookie("casdoor_session_id")
		if ck == nil || ck.Value != "sess-2" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"Please login first"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","data":{"id":"uuid-2","name":"u","email":"u@example.com","emailVerified":false}}`))
	})
	svc, store := newTestInpageService(t, mux)

	_, challenge, err := svc.LoginWithPassword(context.Background(), "u@example.com", "pw")
	if err != nil {
		t.Fatalf("LoginWithPassword: %v", err)
	}
	if challenge == nil || challenge.Token == "" || len(challenge.Props) != 2 || challenge.Props[0].MfaType != "otp" {
		t.Fatalf("challenge = %+v", challenge)
	}
	if len(store.kv) != 1 {
		t.Fatalf("challenge state not stored")
	}

	// 错误验证码:挑战态保留,可重试
	if _, err := svc.CompleteMFA(context.Background(), challenge.Token, "otp", "000000"); err != ErrOIDCMFACodeInvalid {
		t.Fatalf("bad code err = %v, want ErrOIDCMFACodeInvalid", err)
	}
	if len(store.kv) != 1 {
		t.Fatalf("challenge should survive a wrong code")
	}

	// 正确验证码:挑战态消费
	verified, err := svc.CompleteMFA(context.Background(), challenge.Token, "otp", "123456")
	if err != nil {
		t.Fatalf("CompleteMFA: %v", err)
	}
	if verified.ProviderUserID != "uuid-2" {
		t.Fatalf("verified = %+v", verified)
	}
	if len(store.kv) != 0 {
		t.Fatalf("challenge state not consumed after success")
	}
	if _, err := svc.CompleteMFA(context.Background(), challenge.Token, "otp", "123456"); err != ErrOIDCMFAChallengeInvalid {
		t.Fatalf("replay err = %v, want ErrOIDCMFAChallengeInvalid", err)
	}
}

func TestSendRegisterCodeErrors(t *testing.T) {
	mux := http.NewServeMux()
	var calls int
	mux.HandleFunc("/api/send-verification-code", func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.PostForm.Get("method") != "signup" || r.PostForm.Get("captchaType") != "none" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"bad request"}`))
			return
		}
		if strings.Contains(r.PostForm.Get("dest"), "taken") {
			_, _ = w.Write([]byte(`{"status":"error","msg":"Email already exists"}`))
			return
		}
		if strings.Contains(r.PostForm.Get("dest"), "fast") {
			_, _ = w.Write([]byte(`{"status":"error","msg":"you can only send one code in 60s"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	svc, _ := newTestInpageService(t, mux)

	if err := svc.SendRegisterCode(context.Background(), "new@example.com"); err != nil {
		t.Fatalf("SendRegisterCode: %v", err)
	}
	if err := svc.SendRegisterCode(context.Background(), "taken@example.com"); err != ErrOIDCEmailExists {
		t.Fatalf("taken err = %v, want ErrOIDCEmailExists", err)
	}
	if err := svc.SendRegisterCode(context.Background(), "fast@example.com"); err != ErrOIDCCodeResendWait {
		t.Fatalf("fast err = %v, want ErrOIDCCodeResendWait", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestRegisterWithPassword(t *testing.T) {
	mux := http.NewServeMux()
	var signupUsernames []string
	mux.HandleFunc("/api/signup", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		username, _ := body["username"].(string)
		signupUsernames = append(signupUsernames, username)
		if len(signupUsernames) == 1 {
			// 首次尝试的用户名视为已占用,触发冲突重试
			_, _ = w.Write([]byte(`{"status":"error","msg":"The username: ` + username + ` already exists"}`))
			return
		}
		if body["emailCode"] != "654321" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"wrong email code"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "casdoor_session_id", Value: "sess-3"})
		_, _ = w.Write([]byte(`{"status":"ok","data":"kano/` + username + `"}`))
	})
	mux.HandleFunc("/api/get-account", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","data":{"id":"uuid-3","name":"buyer2","email":"buyer2@example.com","emailVerified":true}}`))
	})
	svc, _ := newTestInpageService(t, mux)

	verified, err := svc.RegisterWithPassword(context.Background(), "Buyer2@Example.com", "pw123456", "654321", "")
	if err != nil {
		t.Fatalf("RegisterWithPassword: %v", err)
	}
	if verified.ProviderUserID != "uuid-3" || verified.Email != "buyer2@example.com" {
		t.Fatalf("verified = %+v", verified)
	}
	if len(signupUsernames) != 2 || signupUsernames[0] != "buyer2" || !strings.HasPrefix(signupUsernames[1], "buyer2") || len(signupUsernames[1]) <= len("buyer2") {
		t.Fatalf("username retry = %v", signupUsernames)
	}

	// 验证码错误直接失败,不重试
	if _, err := svc.RegisterWithPassword(context.Background(), "x@example.com", "pw123456", "bad", ""); err != ErrOIDCCodeInvalid {
		t.Fatalf("bad code err = %v, want ErrOIDCCodeInvalid", err)
	}
}

func TestSynthesizeUsername(t *testing.T) {
	cases := map[string]string{
		"john.doe@x.com":  "john_doe",
		"1abc@x.com":      "u1abc",
		"a@x.com":         "ua",
		"中文@x.com":        "user",
		"weird+tag@x.com": "weird_tag",
	}
	for email, want := range cases {
		if got := synthesizeUsername(email, ""); got != want {
			t.Fatalf("synthesizeUsername(%q) = %q, want %q", email, got, want)
		}
	}
	if u := synthesizeUsername("john@x.com", "ab12"); !strings.HasPrefix(u, "john") || !strings.HasSuffix(u, "ab12") {
		t.Fatalf("suffix case = %q", u)
	}
	if u := synthesizeUsername("john@x.com", ""); len(u) > 39 {
		t.Fatalf("too long: %q", u)
	}
	_ = time.Now()
}
