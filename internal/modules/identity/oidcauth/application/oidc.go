package oidcauthapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/golang-jwt/jwt/v5"
)

const (
	oidcStateTTL    = 600 // seconds
	oidcStatePrefix = "oidc:generic:state:"
	oidcReplayTTL   = 300 * time.Second
)

type oidcState struct {
	CodeVerifier string `json:"v"`
	Intent       string `json:"i"`
	UserID       uint   `json:"u"`
}

func base64URLNoPad(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func newPKCEPair() (verifier string, challenge string, err error) {
	buf := make([]byte, 48)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64URLNoPad(buf)
	return verifier, s256Challenge(verifier), nil
}

func s256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64URLNoPad(sum[:])
}

func newState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64URLNoPad(buf), nil
}

// discoveryDocument 返回（带 10 分钟缓存的）OIDC 端点发现文档。
func (s *Service) discoveryDocument(ctx context.Context) (*oidcDiscoveryDocument, error) {
	s.discoveryMu.Lock()
	cached := s.discovery
	stale := time.Since(s.discoveryFetchedAt) > 10*time.Minute
	s.discoveryMu.Unlock()
	if cached != nil && !stale {
		return cached, nil
	}

	cfg := s.currentConfig()
	wellKnown := cfg.Issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCDiscoveryFailed, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCDiscoveryFailed, err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: http %d", ErrOIDCDiscoveryFailed, resp.StatusCode)
	}
	var doc oidcDiscoveryDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOIDCDiscoveryFailed, err)
	}
	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" || doc.JWKSURI == "" {
		return nil, fmt.Errorf("%w: discovery document missing endpoints", ErrOIDCDiscoveryFailed)
	}
	// 端点必须与配置的 issuer 同源，防止发现文档被劫持指向其他主机。
	for _, endpoint := range []string{doc.AuthorizationEndpoint, doc.TokenEndpoint, doc.JWKSURI} {
		if !strings.HasPrefix(endpoint, cfg.Issuer+"/") {
			return nil, fmt.Errorf("%w: endpoint outside issuer origin", ErrOIDCDiscoveryFailed)
		}
	}
	if doc.Issuer != "" && doc.Issuer != cfg.Issuer {
		return nil, fmt.Errorf("%w: issuer mismatch", ErrOIDCDiscoveryFailed)
	}

	s.discoveryMu.Lock()
	s.discovery = &doc
	s.discoveryFetchedAt = time.Now()
	s.discoveryMu.Unlock()
	return &doc, nil
}

// StartOIDCLogin 生成授权 URL，并把 PKCE/state 存入缓存。
func (s *Service) StartOIDCLogin(ctx context.Context, intent string, userID uint) (string, error) {
	if s == nil {
		return "", ErrOIDCAuthConfigInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := s.currentConfig()
	if !cfg.Enabled {
		return "", ErrOIDCAuthDisabled
	}
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.RedirectURI == "" {
		return "", ErrOIDCAuthConfigInvalid
	}
	if intent != LoginIntentLogin && intent != LoginIntentBind {
		return "", ErrOIDCPayloadInvalid
	}
	if s.oidcStateSet == nil {
		return "", ErrOIDCAuthConfigInvalid
	}

	doc, err := s.discoveryDocument(ctx)
	if err != nil {
		return "", err
	}

	state, err := newState()
	if err != nil {
		return "", err
	}
	verifier, challenge, err := newPKCEPair()
	if err != nil {
		return "", err
	}
	rec, err := json.Marshal(oidcState{CodeVerifier: verifier, Intent: intent, UserID: userID})
	if err != nil {
		return "", err
	}
	ok, err := s.oidcStateSet(ctx, oidcStatePrefix+state, string(rec), oidcStateTTL)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrOIDCStateInvalid
	}

	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email")
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return fmt.Sprintf("%s?%s", doc.AuthorizationEndpoint, q.Encode()), nil
}

type oidcJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func parseJWKS(raw []byte) (map[string]*rsa.PublicKey, error) {
	var doc struct {
		Keys []oidcJWK `json:"keys"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if strings.ToUpper(k.Kty) != "RSA" {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.N, "="))
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.E, "="))
		if err != nil {
			continue
		}
		pub := &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(new(big.Int).SetBytes(eb).Int64())}
		if pub.E == 0 {
			continue
		}
		kid := k.Kid
		if kid == "" {
			kid = "_default"
		}
		out[kid] = pub
	}
	if len(out) == 0 {
		return nil, errors.New("no RSA keys in JWKS")
	}
	return out, nil
}

func (s *Service) jwksKey(ctx context.Context, jwksURI, kid string) (*rsa.PublicKey, error) {
	if kid == "" {
		kid = "_default"
	}
	s.jwksMu.Lock()
	cached := s.jwksCache[kid]
	stale := time.Since(s.jwksFetchedAt) > 10*time.Minute
	s.jwksMu.Unlock()
	if cached != nil && !stale {
		return cached, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("jwks http %d", resp.StatusCode)
	}
	keys, err := parseJWKS(body)
	if err != nil {
		return nil, err
	}
	s.jwksMu.Lock()
	s.jwksCache = keys
	s.jwksFetchedAt = time.Now()
	got := s.jwksCache[kid]
	if got == nil && len(keys) == 1 {
		for _, v := range keys {
			got = v
		}
	}
	s.jwksMu.Unlock()
	if got == nil {
		return nil, fmt.Errorf("kid %q not found in JWKS", kid)
	}
	return got, nil
}

type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

type oidcIDClaims struct {
	jwt.RegisteredClaims
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
}

// CompleteOIDCLogin 用授权码换 token 并返回验签后的身份。
func (s *Service) CompleteOIDCLogin(ctx context.Context, code, state string) (*IdentityVerified, string, uint, error) {
	if s == nil {
		return nil, "", 0, ErrOIDCAuthConfigInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	code = strings.TrimSpace(code)
	state = strings.TrimSpace(state)
	if code == "" || state == "" {
		return nil, "", 0, ErrOIDCPayloadInvalid
	}
	cfg := s.currentConfig()
	if !cfg.Enabled {
		return nil, "", 0, ErrOIDCAuthDisabled
	}
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.RedirectURI == "" {
		return nil, "", 0, ErrOIDCAuthConfigInvalid
	}
	if s.oidcStateTake == nil {
		return nil, "", 0, ErrOIDCAuthConfigInvalid
	}

	doc, err := s.discoveryDocument(ctx)
	if err != nil {
		return nil, "", 0, err
	}

	rawState, ok, err := s.oidcStateTake(ctx, oidcStatePrefix+state)
	if err != nil {
		return nil, "", 0, err
	}
	if !ok || rawState == "" {
		return nil, "", 0, ErrOIDCStateInvalid
	}
	var st oidcState
	if err := json.Unmarshal([]byte(rawState), &st); err != nil || st.CodeVerifier == "" {
		return nil, "", 0, ErrOIDCStateInvalid
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURI)
	form.Set("client_id", cfg.ClientID)
	form.Set("code_verifier", st.CodeVerifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, doc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret)))
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("%w: %v", ErrOIDCTokenExchange, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return nil, "", 0, fmt.Errorf("%w: http %d", ErrOIDCTokenExchange, resp.StatusCode)
	}
	var tokResp oidcTokenResponse
	if err := json.Unmarshal(body, &tokResp); err != nil || strings.TrimSpace(tokResp.IDToken) == "" {
		return nil, "", 0, ErrOIDCTokenExchange
	}

	var claims oidcIDClaims
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(cfg.Issuer), jwt.WithAudience(cfg.ClientID), jwt.WithExpirationRequired())
	_, err = parser.ParseWithClaims(tokResp.IDToken, &claims, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		return s.jwksKey(ctx, doc.JWKSURI, kid)
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("%w: %v", ErrOIDCIDTokenInvalid, err)
	}

	providerUserID := strings.TrimSpace(claims.Subject)
	if providerUserID == "" {
		return nil, "", 0, ErrOIDCIDTokenInvalid
	}
	authAt := time.Now()
	if claims.IssuedAt != nil {
		authAt = claims.IssuedAt.Time
	}

	if err := s.markReplay(ctx, providerUserID, hex.EncodeToString(fp8(tokResp.IDToken))); err != nil {
		return nil, "", 0, err
	}

	return &IdentityVerified{
		Provider:       constants.UserOAuthProviderOIDC,
		ProviderUserID: providerUserID,
		Email:          strings.TrimSpace(claims.Email),
		EmailVerified:  claims.EmailVerified,
		Username:       strings.TrimSpace(claims.PreferredUsername),
		DisplayName:    strings.TrimSpace(claims.Name),
		AvatarURL:      strings.TrimSpace(claims.Picture),
		AuthAt:         authAt,
	}, st.Intent, st.UserID, nil
}

func fp8(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:8]
}

func (s *Service) markReplay(ctx context.Context, providerUserID, fingerprint string) error {
	if s == nil {
		return ErrOIDCAuthConfigInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	setNX := s.replaySetNX
	if setNX == nil {
		return ErrOIDCAuthConfigInvalid
	}
	replayKey := "oidc:auth:replay:" + providerUserID + ":" + fingerprint
	ok, err := setNX(ctx, replayKey, "1", oidcReplayTTL)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOIDCReplay
	}
	return nil
}
