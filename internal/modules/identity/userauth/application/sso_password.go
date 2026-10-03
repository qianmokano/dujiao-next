package application

import (
	"context"

	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"
	"github.com/dujiao-next/internal/shared/casdoorcaptcha"
)

func (s *Service) PrepareSSOCaptcha(ctx context.Context, action, account string) (*casdoorcaptcha.Challenge, error) {
	if s.ssoAuthService == nil {
		return nil, ssoauthapp.ErrSSOAuthConfigInvalid
	}
	return s.ssoAuthService.PrepareCaptcha(ctx, action, account)
}

// SSOPasswordLoginInput 页内直连登录输入(凭据由 IdP 校验)。
type SSOPasswordLoginInput struct {
	Account  string // 邮箱或用户名
	Password string
	Captcha  *casdoorcaptcha.Proof
	Context  context.Context
}

// SSOMFAInput 页内直连 MFA 二步输入。
type SSOMFAInput struct {
	Challenge string
	MfaType   string
	Passcode  string
	Context   context.Context
}

// SSORegisterCodeInput 页内注册验证码输入。
type SSORegisterCodeInput struct {
	Email   string
	Captcha *casdoorcaptcha.Proof
	Context context.Context
}

// SSORegisterInput 页内注册输入(成功即登录)。
type SSORegisterInput struct {
	Email       string
	Password    string
	Code        string
	DisplayName string
	Captcha     *casdoorcaptcha.Proof
	Context     context.Context
}

// SSOPasswordLoginResult 页内登录结果:普通登录返回 Login,账号开 MFA 时返回 Challenge。
type SSOPasswordLoginResult struct {
	Login     *UserLoginResult
	Challenge *ssoauthapp.MFAChallenge
}

// LoginWithSSOPassword 页内直连登录:凭据交 IdP 校验,成功后完成本站身份关联和登录。
func (s *Service) LoginWithSSOPassword(input SSOPasswordLoginInput) (*SSOPasswordLoginResult, error) {
	if s.ssoAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, ssoauthapp.ErrSSOAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, challenge, err := s.ssoAuthService.LoginWithPassword(ctx, input.Account, input.Password, input.Captcha)
	if err != nil {
		return nil, err
	}
	if challenge != nil {
		return &SSOPasswordLoginResult{Challenge: challenge}, nil
	}
	result, err := s.loginVerifiedSSO(verified)
	if err != nil {
		return nil, err
	}
	return &SSOPasswordLoginResult{Login: result}, nil
}

// CompleteSSOMFA 续接页内登录的 MFA 二步验证。
func (s *Service) CompleteSSOMFA(input SSOMFAInput) (*UserLoginResult, error) {
	if s.ssoAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, ssoauthapp.ErrSSOAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.ssoAuthService.CompleteMFA(ctx, input.Challenge, input.MfaType, input.Passcode)
	if err != nil {
		return nil, err
	}
	return s.loginVerifiedSSO(verified)
}

// SendSSORegisterCode 页内注册:让 IdP 发送邮箱验证码。
func (s *Service) SendSSORegisterCode(input SSORegisterCodeInput) error {
	if s.ssoAuthService == nil {
		return ssoauthapp.ErrSSOAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return s.ssoAuthService.SendRegisterCode(ctx, input.Email, input.Captcha)
}

// RegisterWithSSO 页内注册:IdP 侧完成邮箱验证码开户,成功即登录本站。
func (s *Service) RegisterWithSSO(input SSORegisterInput) (*UserLoginResult, error) {
	if s.ssoAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, ssoauthapp.ErrSSOAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.ssoAuthService.RegisterWithPassword(ctx, input.Email, input.Password, input.Code, input.DisplayName, input.Captcha)
	if err != nil {
		return nil, err
	}
	return s.loginVerifiedSSO(verified)
}
