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

	ErrOIDCInvalidCredentials  = errors.New("oidc invalid credentials")
	ErrOIDCAccountFrozen       = errors.New("oidc account frozen")
	ErrOIDCCaptchaRequired     = errors.New("oidc captcha required")
	ErrOIDCMFARequired         = errors.New("oidc mfa required")
	ErrOIDCMFAChallengeInvalid = errors.New("oidc mfa challenge invalid")
	ErrOIDCMFACodeInvalid      = errors.New("oidc mfa code invalid")
	ErrOIDCCodeInvalid         = errors.New("oidc email code invalid")
	ErrOIDCEmailExists         = errors.New("oidc email already exists")
	ErrOIDCCodeResendWait      = errors.New("oidc code resend wait")
	ErrOIDCRemoteRejected      = errors.New("oidc remote rejected")
)
