package oidcauthcache

import (
	"context"
	"time"

	"github.com/dujiao-next/internal/cache"
	oidcauthapp "github.com/dujiao-next/internal/modules/identity/oidcauth/application"
)

func Options() []oidcauthapp.Option {
	return []oidcauthapp.Option{
		oidcauthapp.WithReplaySetNX(cache.SetNX),
		oidcauthapp.WithOIDCStateStore(setOIDCState, takeOIDCState),
	}
}

func setOIDCState(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	return cache.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second)
}

func takeOIDCState(ctx context.Context, key string) (string, bool, error) {
	value, err := cache.GetString(ctx, key)
	if err != nil {
		return "", false, err
	}
	if value == "" {
		return "", false, nil
	}
	_ = cache.Del(ctx, key)
	return value, true, nil
}
