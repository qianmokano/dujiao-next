package ssoauthapp

import (
	"errors"

	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
)

var (
	ErrSSOAuthDisabled      = errors.New("sso auth disabled")
	ErrSSOAuthConfigInvalid = settingssecurity.ErrSSOAuthConfigInvalid
	ErrSSOPayloadInvalid    = errors.New("sso auth payload invalid")

	ErrSSOInvalidCredentials  = errors.New("sso invalid credentials")
	ErrSSOAccountFrozen       = errors.New("sso account frozen")
	ErrSSOCaptchaRequired     = errors.New("sso captcha required")
	ErrSSOMFARequired         = errors.New("sso mfa required")
	ErrSSOMFAChallengeInvalid = errors.New("sso mfa challenge invalid")
	ErrSSOMFACodeInvalid      = errors.New("sso mfa code invalid")
	ErrSSOCodeInvalid         = errors.New("sso email code invalid")
	ErrSSOEmailExists         = errors.New("sso email already exists")
	ErrSSOCodeResendWait      = errors.New("sso code resend wait")
	ErrSSORemoteRejected      = errors.New("sso remote rejected")
)
