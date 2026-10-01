package application

import (
	"context"

	oidcauthapp "github.com/dujiao-next/internal/modules/identity/oidcauth/application"
)

// OIDCPasswordLoginInput 页内直连登录输入(凭据由 IdP 校验)。
type OIDCPasswordLoginInput struct {
	Account  string // 邮箱或用户名
	Password string
	Context  context.Context
}

// OIDCMFAInput 页内直连 MFA 二步输入。
type OIDCMFAInput struct {
	Challenge string
	MfaType   string
	Passcode  string
	Context   context.Context
}

// OIDCRegisterCodeInput 页内注册验证码输入。
type OIDCRegisterCodeInput struct {
	Email   string
	Context context.Context
}

// OIDCRegisterInput 页内注册输入(成功即登录)。
type OIDCRegisterInput struct {
	Email       string
	Password    string
	Code        string
	DisplayName string
	Context     context.Context
}

// OIDCPasswordLoginResult 页内登录结果:普通登录返回 Login,账号开 MFA 时返回 Challenge。
type OIDCPasswordLoginResult struct {
	Login     *UserLoginResult
	Challenge *oidcauthapp.MFAChallenge
}

// LoginWithOIDCPassword 页内直连登录:凭据交 IdP 校验,成功后复用 OIDC 登录收尾。
func (s *Service) LoginWithOIDCPassword(input OIDCPasswordLoginInput) (*OIDCPasswordLoginResult, error) {
	if s.oidcAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, challenge, err := s.oidcAuthService.LoginWithPassword(ctx, input.Account, input.Password)
	if err != nil {
		return nil, err
	}
	if challenge != nil {
		return &OIDCPasswordLoginResult{Challenge: challenge}, nil
	}
	result, err := s.loginVerifiedOIDC(verified)
	if err != nil {
		return nil, err
	}
	return &OIDCPasswordLoginResult{Login: result}, nil
}

// CompleteOIDCMFA 续接页内登录的 MFA 二步验证。
func (s *Service) CompleteOIDCMFA(input OIDCMFAInput) (*UserLoginResult, error) {
	if s.oidcAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.oidcAuthService.CompleteMFA(ctx, input.Challenge, input.MfaType, input.Passcode)
	if err != nil {
		return nil, err
	}
	return s.loginVerifiedOIDC(verified)
}

// SendOIDCRegisterCode 页内注册:让 IdP 发送邮箱验证码。
func (s *Service) SendOIDCRegisterCode(input OIDCRegisterCodeInput) error {
	if s.oidcAuthService == nil {
		return oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return s.oidcAuthService.SendRegisterCode(ctx, input.Email)
}

// RegisterWithOIDC 页内注册:IdP 侧完成邮箱验证码开户,成功即登录本站。
func (s *Service) RegisterWithOIDC(input OIDCRegisterInput) (*UserLoginResult, error) {
	if s.oidcAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.oidcAuthService.RegisterWithPassword(ctx, input.Email, input.Password, input.Code, input.DisplayName)
	if err != nil {
		return nil, err
	}
	return s.loginVerifiedOIDC(verified)
}
