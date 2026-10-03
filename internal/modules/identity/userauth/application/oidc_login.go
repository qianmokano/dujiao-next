package application

import (
	"context"
	"fmt"
	"net/url"
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
	if input.Intent == oidcauthapp.LoginIntentBind {
		if err := s.requireLocalIdentityManagement(); err != nil {
			return "", err
		}
	}
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
	if err := s.requireLocalIdentityManagement(); err != nil {
		return nil, err
	}
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
	if verified == nil || verified.Provider == "" || strings.TrimSpace(verified.ProviderUserID) == "" {
		return nil, oidcauthapp.ErrOIDCPayloadInvalid
	}
	if s.authUnitOfWork == nil {
		return nil, oidcauthapp.ErrOIDCAuthConfigInvalid
	}
	for attempt := 0; attempt < 2; attempt++ {
		before, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
		if err != nil {
			return nil, err
		}
		var user *userdomain.User
		var created bool
		err = s.authUnitOfWork.WithinTransaction(context.Background(), func(tx AuthTransaction) error {
			// Lock the user before the identity, matching other authentication transactions.
			var identity *externalidentitydomain.Identity
			var txErr error
			if before != nil {
				user, txErr = activeTransactionUser(tx, before.UserID)
				if txErr != nil {
					return txErr
				}
				identity, txErr = tx.GetIdentityByProviderUserID(verified.Provider, verified.ProviderUserID)
				if txErr != nil {
					return txErr
				}
				if identity == nil || identity.UserID != user.ID {
					return errGoogleLoginMappingChanged
				}
			} else {
				user, created, txErr = s.findOrCreateOIDCUser(tx, verified)
				if txErr != nil {
					return txErr
				}
				identity, txErr = tx.GetIdentityByProviderUserID(verified.Provider, verified.ProviderUserID)
				if txErr != nil {
					return txErr
				}
				if identity != nil {
					return errGoogleLoginMappingChanged
				}
				identity = &externalidentitydomain.Identity{UserID: user.ID, Provider: verified.Provider, ProviderUserID: verified.ProviderUserID, CreatedAt: time.Now()}
				applyOIDCIdentity(verified, identity)
				identity.UpdatedAt = time.Now()
				if txErr = tx.CreateIdentity(identity); txErr != nil {
					return txErr
				}
			}
			if applyOIDCIdentity(verified, identity) {
				identity.UpdatedAt = time.Now()
				if txErr = tx.UpdateIdentity(identity); txErr != nil {
					return txErr
				}
			}
			fields := map[string]interface{}{}
			if s.oidcAuthService.OnlyEnabled() {
				user.DisplayName = resolveOIDCDisplayName(verified, user.Email)
				fields["display_name"] = user.DisplayName
			}
			email, emailErr := normalizeUserSuppliedEmail(verified.Email)
			if user.EmailVerifiedAt == nil && verified.EmailVerified && emailErr == nil && email == strings.ToLower(strings.TrimSpace(user.Email)) {
				now := time.Now()
				user.EmailVerifiedAt = &now
				fields["email_verified_at"] = now
			}
			if len(fields) == 0 {
				return nil
			}
			user.UpdatedAt = time.Now()
			fields["updated_at"] = user.UpdatedAt
			return tx.UpdateUserFields(user.ID, fields)
		})
		if err != nil {
			// A concurrent first login may have established the same mapping.
			if attempt == 0 {
				latest, lookupErr := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
				if lookupErr == nil && latest != nil && (before == nil || err == errGoogleLoginMappingChanged) {
					continue
				}
			}
			return nil, err
		}
		if created && s.memberLevelSvc != nil {
			_ = s.memberLevelSvc.AssignDefaultLevel(user.ID)
			if refreshed, refreshErr := s.userRepo.GetByID(user.ID); refreshErr == nil && refreshed != nil {
				user = refreshed
			}
		}
		return s.completeExternalLogin(user, constants.LoginLogSourceOIDC)
	}
	return nil, errGoogleLoginMappingChanged
}

// findOrCreateOIDCUser 按邮箱找已有用户；不存在则直接建号。
// SSO 是本部署唯一的注册入口：IdP（自有 Casdoor）注册时已强制邮箱验证码，
// 因此这里有意不检查站点注册开关——关闭本地注册不影响经 IdP 的新用户建号。
// 只有已验证邮箱可以首次关联；不同 subject 不得覆盖已有同提供方绑定。
func (s *Service) findOrCreateOIDCUser(tx AuthTransaction, verified *oidcauthapp.IdentityVerified) (*userdomain.User, bool, error) {
	email, err := normalizeUserSuppliedEmail(verified.Email)
	if err != nil || !verified.EmailVerified {
		return nil, false, ErrEmailNotVerified
	}

	user, err := tx.GetUserByEmail(email)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		if strings.ToLower(strings.TrimSpace(user.Status)) != constants.UserStatusActive {
			return nil, false, ErrUserDisabled
		}
		bound, err := tx.GetIdentityByUserProvider(user.ID, verified.Provider)
		if err != nil {
			return nil, false, err
		}
		if bound != nil && bound.ProviderUserID != verified.ProviderUserID {
			return nil, false, ErrUserOAuthIdentityExists
		}
		return user, false, nil
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
		EmailVerifiedAt:       &now,
		LastLoginAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := tx.CreateUser(user); err != nil {
		return nil, false, err
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
	avatar, present := oidcAvatar(verified)
	if present && identity.AvatarURL != avatar {
		identity.AvatarURL = avatar
		changed = true
	}
	if identity.AuthAt == nil || !identity.AuthAt.Equal(verified.AuthAt) {
		authAt := verified.AuthAt
		identity.AuthAt = &authAt
		changed = true
	}
	return changed
}

func oidcAvatar(verified *oidcauthapp.IdentityVerified) (string, bool) {
	value := strings.TrimSpace(verified.AvatarURL)
	if value == "" {
		return "", verified.AvatarPresent
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return "", false
	}
	return value, true
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
	if err := s.requireLocalIdentityManagement(); err != nil {
		return err
	}
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
