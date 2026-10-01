package oidcauthapp

import (
	"errors"

	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
)

var (
	ErrOIDCAuthDisabled      = errors.New("oidc auth disabled")
	ErrOIDCAuthConfigInvalid = settingssecurity.ErrOIDCAuthConfigInvalid
	ErrOIDCPayloadInvalid    = errors.New("oidc auth payload invalid")
	ErrOIDCStateInvalid      = errors.New("oidc state invalid")
	ErrOIDCDiscoveryFailed   = errors.New("oidc discovery failed")
	ErrOIDCTokenExchange     = errors.New("oidc token exchange failed")
	ErrOIDCIDTokenInvalid    = errors.New("oidc id token invalid")
	ErrOIDCReplay            = errors.New("oidc auth replay")
)
