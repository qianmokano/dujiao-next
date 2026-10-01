package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"

	"github.com/dujiao-next/internal/constants"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	oidcauthapp "github.com/dujiao-next/internal/modules/identity/oidcauth/application"

	"golang.org/x/crypto/bcrypt"
)

// StartOIDCInput 启动通用 OIDC 流程输入
type StartOIDCInput struct {
	Intent  string // "login" | "bind"
	UserID  uint
	Context context.Context
}

// LoginWithOIDCInput 通用 OIDC 登录输入
type LoginWithOIDCInput struct {
	Code    string
	State   string
	Context context.Context
}

// BindOIDCInput 通用 OIDC 绑定输入
type BindOIDCInput struct {
	UserID  uint
	Code    string
	State   string
	Context context.Context
}

// StartOIDC 生成通用 OIDC 授权 URL
func (s *Service) StartOIDC(input StartOIDCInput) (string, error) {
	if s.oidcAuthService == nil {
		return "", oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	intent := input.Intent
	if intent != oidcauthapp.LoginIntentLogin && intent != oidcauthapp.LoginIntentBind {
		return "", oidcauthapp.ErrOIDCPayloadInvalid
	}
	return s.oidcAuthService.StartOIDCLogin(ctx, intent, input.UserID)
}

// LoginWithOIDC 通过通用 OIDC 回调登录
func (s *Service) LoginWithOIDC(input LoginWithOIDCInput) (*UserLoginResult, error) {
	if s.oidcAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, intent, _, err := s.oidcAuthService.CompleteOIDCLogin(ctx, input.Code, input.State)
	if err != nil {
		return nil, err
	}
	if intent != oidcauthapp.LoginIntentLogin {
		return nil, oidcauthapp.ErrOIDCPayloadInvalid
	}
	return s.loginVerifiedOIDC(verified)
}

// BindOIDC 通过通用 OIDC 回调绑定当前用户
func (s *Service) BindOIDC(input BindOIDCInput) (*externalidentitydomain.Identity, error) {
	if input.UserID == 0 {
		return nil, ErrNotFound
	}
	if s.oidcAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, intent, userID, err := s.oidcAuthService.CompleteOIDCLogin(ctx, input.Code, input.State)
	if err != nil {
		return nil, err
	}
	if intent != oidcauthapp.LoginIntentBind || userID != input.UserID {
		return nil, oidcauthapp.ErrOIDCPayloadInvalid
	}
	return s.bindVerifiedOIDC(input.UserID, verified)
}

// loginVerifiedOIDC 完成通用 OIDC 登录：已有绑定则落回原账号，否则按邮箱关联/建号。
func (s *Service) loginVerifiedOIDC(verified *oidcauthapp.IdentityVerified) (*UserLoginResult, error) {
	identity, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
	if err != nil {
		return nil, err
	}

	if identity != nil {
		user, err := s.getActiveUserByID(identity.UserID)
		if err != nil {
			return nil, err
		}
		if applyOIDCIdentity(verified, identity) {
			identity.UpdatedAt = time.Now()
			if err := s.userOAuthIdentityRepo.Update(identity); err != nil {
				return nil, err
			}
		}
		return s.completeExternalLogin(user, constants.LoginLogSourceOIDC)
	}

	user, _, err := s.findOrCreateOIDCUser(verified)
	if err != nil {
		return nil, err
	}
	identity = &externalidentitydomain.Identity{
		UserID:         user.ID,
		Provider:       verified.Provider,
		ProviderUserID: verified.ProviderUserID,
		Username:       verified.Username,
		AvatarURL:      verified.AvatarURL,
		AuthAt:         &verified.AuthAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.userOAuthIdentityRepo.Create(identity); err != nil {
		// 并发下同一身份可能已被另一个请求绑定，落回已有绑定。
		existing, getErr := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
		if getErr != nil || existing == nil {
			return nil, err
		}
		identity = existing
		user, err = s.getActiveUserByID(existing.UserID)
		if err != nil {
			return nil, err
		}
		return s.completeExternalLogin(user, constants.LoginLogSourceOIDC)
	}
	return s.completeExternalLogin(user, constants.LoginLogSourceOIDC)
}

// findOrCreateOIDCUser 按邮箱找已有用户；不存在则按注册策略建号。
// 邮箱在 IdP 侧是权威身份（本部署为自有 Casdoor），首登即与同邮箱本地账号合并。
func (s *Service) findOrCreateOIDCUser(verified *oidcauthapp.IdentityVerified) (*userdomain.User, bool, error) {
	email := strings.TrimSpace(verified.Email)
	if email == "" {
		email = buildOIDCPlaceholderEmail(verified.ProviderUserID)
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		if strings.ToLower(strings.TrimSpace(user.Status)) != constants.UserStatusActive {
			return nil, false, ErrUserDisabled
		}
		return user, false, nil
	}

	if s.settingService != nil {
		registrationEnabled, err := s.settingService.GetRegistrationEnabled(true)
		if err != nil {
			return nil, false, err
		}
		if !registrationEnabled {
			return nil, false, ErrRegistrationDisabled
		}
	}

	randomSuffix, err := randomNumericCode(16)
	if err != nil {
		return nil, false, err
	}
	passwordSeed := fmt.Sprintf("oidc_%s_%s", verified.ProviderUserID, randomSuffix)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordSeed), bcrypt.DefaultCost)
	if err != nil {
		return nil, false, err
	}

	now := time.Now()
	user = &userdomain.User{
		Email:                 email,
		PasswordHash:          string(hashedPassword),
		PasswordSetupRequired: true,
		DisplayName:           resolveOIDCDisplayName(verified, email),
		Status:                constants.UserStatusActive,
		LastLoginAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.userRepo.Create(user); err != nil {
		// 同邮箱注册竞争：落回已存在用户。
		existing, getErr := s.userRepo.GetByEmail(email)
		if getErr != nil || existing == nil {
			return nil, false, err
		}
		if strings.ToLower(strings.TrimSpace(existing.Status)) != constants.UserStatusActive {
			return nil, false, ErrUserDisabled
		}
		return existing, false, nil
	}
	if s.memberLevelSvc != nil {
		_ = s.memberLevelSvc.AssignDefaultLevel(user.ID)
		// 同步内存对象的等级，避免调用方后续 Update(Save) 用零值覆盖数据库
		if refreshed, err := s.userRepo.GetByID(user.ID); err == nil && refreshed != nil {
			user.MemberLevelID = refreshed.MemberLevelID
		}
	}
	return user, true, nil
}

func resolveOIDCDisplayName(verified *oidcauthapp.IdentityVerified, email string) string {
	for _, candidate := range []string{verified.DisplayName, verified.Username} {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}

func buildOIDCPlaceholderEmail(providerUserID string) string {
	trimmed := strings.TrimSpace(providerUserID)
	if trimmed == "" {
		trimmed = "unknown"
	}
	return fmt.Sprintf("oidc_%s@login.local", trimmed)
}

func applyOIDCIdentity(verified *oidcauthapp.IdentityVerified, identity *externalidentitydomain.Identity) bool {
	if verified == nil || identity == nil {
		return false
	}
	changed := false
	if identity.Username != verified.Username {
		identity.Username = verified.Username
		changed = true
	}
	if identity.AvatarURL != verified.AvatarURL {
		identity.AvatarURL = verified.AvatarURL
		changed = true
	}
	if identity.AuthAt == nil || !identity.AuthAt.Equal(verified.AuthAt) {
		authAt := verified.AuthAt
		identity.AuthAt = &authAt
		changed = true
	}
	return changed
}

// OIDCBinding 通用 OIDC 绑定视图。
type OIDCBinding struct {
	Identity  *externalidentitydomain.Identity
	CanUnbind bool
}

func (s *Service) bindVerifiedOIDC(userID uint, verified *oidcauthapp.IdentityVerified) (*externalidentitydomain.Identity, error) {
	if _, err := s.getActiveUserByID(userID); err != nil {
		return nil, err
	}

	occupied, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
	if err != nil {
		return nil, err
	}
	if occupied != nil && occupied.UserID != userID {
		return nil, ErrUserOAuthIdentityExists
	}

	current, err := s.userOAuthIdentityRepo.GetByUserProvider(userID, verified.Provider)
	if err != nil {
		return nil, err
	}
	if current != nil && current.ProviderUserID != verified.ProviderUserID {
		return nil, ErrUserOAuthAlreadyBound
	}
	if current == nil {
		current = &externalidentitydomain.Identity{
			UserID:         userID,
			Provider:       verified.Provider,
			ProviderUserID: verified.ProviderUserID,
			Username:       verified.Username,
			AvatarURL:      verified.AvatarURL,
			AuthAt:         &verified.AuthAt,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.userOAuthIdentityRepo.Create(current); err != nil {
			occupied, occupiedErr := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
			if occupiedErr == nil && occupied != nil && occupied.UserID != userID {
				return nil, ErrUserOAuthIdentityExists
			}
			latest, latestErr := s.userOAuthIdentityRepo.GetByUserProvider(userID, verified.Provider)
			if latestErr == nil && latest != nil {
				if latest.ProviderUserID != verified.ProviderUserID {
					return nil, ErrUserOAuthAlreadyBound
				}
				return latest, nil
			}
			return nil, err
		}
		return current, nil
	}

	if applyOIDCIdentity(verified, current) {
		current.UpdatedAt = time.Now()
		if err := s.userOAuthIdentityRepo.Update(current); err != nil {
			return nil, err
		}
	}
	return current, nil
}

// UnbindOIDC 解绑通用 OIDC
func (s *Service) UnbindOIDC(userID uint) error {
	if err := s.unbindExternalIdentity(userID, constants.UserOAuthProviderOIDC); err != nil {
		if err == errExternalIdentityUnbindLocked {
			return ErrOIDCUnbindRequiresLocalLogin
		}
		return err
	}
	return nil
}

// GetOIDCBinding 获取通用 OIDC 绑定
func (s *Service) GetOIDCBinding(userID uint) (*OIDCBinding, error) {
	if userID == 0 {
		return nil, ErrNotFound
	}
	if s.userOAuthIdentityRepo == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	user, err := s.getActiveUserByID(userID)
	if err != nil {
		return nil, err
	}
	identity, err := s.userOAuthIdentityRepo.GetByUserProvider(userID, constants.UserOAuthProviderOIDC)
	if err != nil {
		return nil, err
	}
	result := &OIDCBinding{Identity: identity}
	if identity == nil {
		return result, nil
	}
	result.CanUnbind, err = s.canUnbindExternalIdentity(user, identity)
	if err != nil {
		return nil, err
	}
	return result, nil
}
