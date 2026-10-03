package ssoauthcache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dujiao-next/internal/cache"
	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"
)

func Options() []ssoauthapp.Option {
	return []ssoauthapp.Option{
		ssoauthapp.WithCaptchaStore(captchaStore{}),
		ssoauthapp.WithMFAChallengeStore(setMFAChallenge, getMFAChallenge, delMFAChallenge),
	}
}

type captchaStore struct{}

func (captchaStore) Set(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if !cache.Enabled() {
		return false, cache.ErrUnavailable
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return cache.SetNX(ctx, key, string(payload), ttl)
}

func (captchaStore) Take(ctx context.Context, key string) (string, bool, error) {
	var value string
	ok, err := cache.GetDelJSONRequired(ctx, key, &value)
	return value, ok, err
}

func setMFAChallenge(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	if !cache.Enabled() {
		return false, cache.ErrUnavailable
	}
	return cache.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second)
}

func getMFAChallenge(ctx context.Context, key string) (string, bool, error) {
	value, err := cache.GetString(ctx, key)
	if err != nil {
		return "", false, err
	}
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func delMFAChallenge(ctx context.Context, key string) error {
	return cache.Del(ctx, key)
}
