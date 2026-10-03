package oidcauthapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/golang-jwt/jwt/v5"
)

func newTestOIDCService(t *testing.T, issuer string) (*Service, *map[string]string) {
	t.Helper()
	svc := NewService(config.OIDCAuthConfig{
		Enabled:      true,
		Issuer:       issuer,
		ClientID:     "dujiao-client",
		ClientSecret: "topsecret",
		RedirectURI:  "https://shop.example.com/auth/oidc/callback",
		DisplayName:  "统一登录",
	})
	store := map[string]string{}
	svc.oidcStateSet = func(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
		if _, ok := store[key]; ok {
			return false, nil
		}
		store[key] = value
		return true, nil
	}
	svc.oidcStateTake = func(ctx context.Context, key string) (string, bool, error) {
		v, ok := store[key]
		if ok {
			delete(store, key)
		}
		return v, ok, nil
	}
	replay := map[string]bool{}
	svc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		if replay[key] {
			return false, nil
		}
		replay[key] = true
		return true, nil
	}
	return svc, &store
}

func TestStartOIDCLoginDisabled(t *testing.T) {
	svc, _ := newTestOIDCService(t, "https://idp.example.com")
	svc.SetConfig(config.OIDCAuthConfig{Enabled: false, Issuer: "https://idp.example.com", ClientID: "c", ClientSecret: "s", RedirectURI: "https://shop.example.com/auth/oidc/callback"})
	if _, err := svc.StartOIDCLogin(context.Background(), LoginIntentLogin, 0); err != ErrOIDCAuthDisabled {
		t.Fatalf("err = %v, want ErrOIDCAuthDisabled", err)
	}
}

func TestDiscoveryEndpointsOutsideIssuerRejected(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 "https://idp.example.com",
			"authorization_endpoint": "https://evil.example.com/auth",
			"token_endpoint":         "https://idp.example.com/token",
			"jwks_uri":               "https://idp.example.com/jwks",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc, _ := newTestOIDCService(t, srv.URL)
	if _, err := svc.StartOIDCLogin(context.Background(), LoginIntentLogin, 0); err == nil {
		t.Fatalf("expected discovery rejection for foreign endpoint")
	}
}

func TestStartAndCompleteOIDCLogin(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "test-kid-1"
	const clientID = "dujiao-client"
	const subject = "user-abc-123"

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base := srv.URL
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 base,
			"authorization_endpoint": base + "/login/oauth/authorize",
			"token_endpoint":         base + "/api/login/oauth/access_token",
			"jwks_uri":               base + "/api/certs",
		})
	})
	mux.HandleFunc("/api/certs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": kid,
			"n": base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
		}}})
	})

	var gotVerifier string
	profileClaims := jwt.MapClaims{
		"email_verified": true, "name": "测试买家", "preferred_username": "buyer",
		"picture": "https://idp.example.com/avatar.png",
	}
	mux.HandleFunc("/api/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "authcode-1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gotVerifier = r.Form.Get("code_verifier")
		if gotVerifier == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		now := time.Now()
		claims := jwt.MapClaims{
			"iss":   base,
			"aud":   clientID,
			"sub":   subject,
			"iat":   now.Unix(),
			"exp":   now.Add(time.Hour).Unix(),
			"email": "buyer@example.com",
		}
		for key, value := range profileClaims {
			claims[key] = value
		}
		idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		idToken.Header["kid"] = kid
		signed, signErr := idToken.SignedString(priv)
		if signErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "at", "id_token": signed, "token_type": "Bearer"})
	})

	svc, _ := newTestOIDCService(t, base)

	authURL, err := svc.StartOIDCLogin(context.Background(), LoginIntentLogin, 0)
	if err != nil {
		t.Fatalf("StartOIDCLogin: %v", err)
	}
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("bad url: %v", err)
	}
	if u.Path != "/login/oauth/authorize" {
		t.Fatalf("unexpected authorize path: %s", u.Path)
	}
	q := u.Query()
	if q.Get("client_id") != clientID {
		t.Fatalf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("scope") != "openid profile email" {
		t.Fatalf("scope = %q", q.Get("scope"))
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q", q.Get("code_challenge_method"))
	}

	verified, intent, userID, err := svc.CompleteOIDCLogin(context.Background(), "authcode-1", q.Get("state"))
	if err != nil {
		t.Fatalf("CompleteOIDCLogin: %v", err)
	}
	if intent != LoginIntentLogin || userID != 0 {
		t.Fatalf("intent/userID = %q/%d", intent, userID)
	}
	if verified.Provider != constants.UserOAuthProviderOIDC || verified.ProviderUserID != subject {
		t.Fatalf("verified = %+v", verified)
	}
	if verified.Email != "buyer@example.com" || !verified.EmailVerified {
		t.Fatalf("email claims = %+v", verified)
	}
	if verified.DisplayName != "测试买家" || verified.Username != "buyer" {
		t.Fatalf("profile claims = %+v", verified)
	}
	if gotVerifier == "" {
		t.Fatalf("code_verifier not sent to token endpoint")
	}

	// A subsequent callback from Casdoor's default format must retain the display name,
	// avatar and verified email instead of replacing the nickname with the username.
	profileClaims = jwt.MapClaims{
		"name": "buyer", "displayName": "更新后的昵称", "emailVerified": true,
		"avatar": "https://idp.example.com/updated-avatar.png",
	}
	authURL, err = svc.StartOIDCLogin(context.Background(), LoginIntentLogin, 0)
	if err != nil {
		t.Fatal(err)
	}
	u, err = url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	verified, _, _, err = svc.CompleteOIDCLogin(context.Background(), "authcode-1", u.Query().Get("state"))
	if err != nil {
		t.Fatal(err)
	}
	if verified.DisplayName != "更新后的昵称" || verified.Username != "buyer" || !verified.EmailVerified || verified.AvatarURL != "https://idp.example.com/updated-avatar.png" || !verified.AvatarPresent {
		t.Fatalf("Casdoor profile claims = %+v", verified)
	}
}

func TestOIDCProfileClaimsFormatsAndFieldPresence(t *testing.T) {
	for _, tc := range []struct {
		name, payload, displayName, username, avatar string
		avatarPresent, emailVerified                 bool
	}{
		{"standard", `{"name":" Standard Buyer ","preferred_username":" buyer ","picture":"https://idp.example/avatar.png","email_verified":true}`, "Standard Buyer", "buyer", "https://idp.example/avatar.png", true, true},
		{"casdoor", `{"name":" buyer ","displayName":" Passport Buyer ","avatar":"https://idp.example/passport.png","emailVerified":true}`, "Passport Buyer", "buyer", "https://idp.example/passport.png", true, true},
		{"casdoor empty display name", `{"name":"buyer","displayName":" "}`, "", "buyer", "", false, false},
		{"explicit username retained", `{"name":"buyer","displayName":"Passport Buyer","preferred_username":"subject-name"}`, "Passport Buyer", "subject-name", "", false, false},
		{"avatar missing", `{"name":"Buyer"}`, "Buyer", "", "", false, false},
		{"avatar null", `{"picture":null,"avatar":null}`, "", "", "", false, false},
		{"standard avatar clear", `{"picture":"","avatar":"https://idp.example/old.png"}`, "", "", "", true, false},
		{"casdoor avatar clear", `{"avatar":""}`, "", "", "", true, false},
		{"standard avatar priority", `{"picture":"https://idp.example/standard.png","avatar":"https://idp.example/old.png"}`, "", "", "https://idp.example/standard.png", true, false},
		{"standard unverified priority", `{"email_verified":false,"emailVerified":true}`, "", "", "", false, false},
		{"casdoor unverified", `{"emailVerified":false}`, "", "", "", false, false},
		{"null standard verification", `{"email_verified":null,"emailVerified":true}`, "", "", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var claims oidcIDClaims
			if err := json.Unmarshal([]byte(tc.payload), &claims); err != nil {
				t.Fatal(err)
			}
			claims.Subject = " fixture-subject "
			claims.Email = " buyer@example.com "
			authAt := time.Unix(1700000000, 0)
			identity := claims.identity(authAt)
			if identity.DisplayName != tc.displayName || identity.Username != tc.username || identity.AvatarURL != tc.avatar || identity.AvatarPresent != tc.avatarPresent || identity.EmailVerified != tc.emailVerified {
				t.Fatalf("identity = %+v", identity)
			}
			if identity.ProviderUserID != "fixture-subject" || identity.Email != "buyer@example.com" || identity.Provider != constants.UserOAuthProviderOIDC || !identity.AuthAt.Equal(authAt) {
				t.Fatalf("identity metadata = %+v", identity)
			}
		})
	}
}

func TestCompleteOIDCLoginStateSingleUse(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "test-kid-1"
	const clientID = "dujiao-client"

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base := srv.URL
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": base + "/authorize",
			"token_endpoint":         base + "/token",
			"jwks_uri":               base + "/certs",
		})
	})
	mux.HandleFunc("/certs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": kid,
			"n": base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss": base, "aud": clientID, "sub": "s1", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		})
		idToken.Header["kid"] = kid
		signed, _ := idToken.SignedString(priv)
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": signed})
	})

	svc, _ := newTestOIDCService(t, base)
	authURL, err := svc.StartOIDCLogin(context.Background(), LoginIntentLogin, 0)
	if err != nil {
		t.Fatalf("StartOIDCLogin: %v", err)
	}
	u, _ := url.Parse(authURL)
	state := u.Query().Get("state")

	// state 一次性消费:第一次成功,第二次拒绝
	if _, _, _, err := svc.CompleteOIDCLogin(context.Background(), "code-1", state); err != nil {
		t.Fatalf("first complete: %v", err)
	}
	if _, _, _, err := svc.CompleteOIDCLogin(context.Background(), "code-1", state); err != ErrOIDCStateInvalid {
		t.Fatalf("second complete err = %v, want ErrOIDCStateInvalid", err)
	}
}

func TestPublicConfigRequiresFullEndpoints(t *testing.T) {
	svc := NewService(config.OIDCAuthConfig{Enabled: true, Issuer: "https://idp.example.com", ClientID: "c"})
	public := svc.PublicConfig()
	if enabled, _ := public["enabled"].(bool); enabled {
		t.Fatalf("enabled should be false when secret/redirect missing")
	}
	svc.SetConfig(config.OIDCAuthConfig{Enabled: true, Issuer: "https://idp.example.com", ClientID: "c", ClientSecret: "s", RedirectURI: "https://shop.example.com/auth/oidc/callback"})
	public = svc.PublicConfig()
	if enabled, _ := public["enabled"].(bool); !enabled {
		t.Fatalf("enabled should be true with full endpoints")
	}
	if name, _ := public["display_name"].(string); name != "" {
		t.Fatalf("display_name = %q", name)
	}
}
