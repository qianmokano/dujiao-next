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
	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"

	"golang.org/x/crypto/bcrypt"
)

// loginVerifiedSSO 完成通行证登录：已有绑定则落回原账号，否则按邮箱关联/建号。
func (s *Service) loginVerifiedSSO(verified *ssoauthapp.IdentityVerified) (*UserLoginResult, error) {
	if verified == nil || verified.Provider == "" || strings.TrimSpace(verified.ProviderUserID) == "" {
		return nil, ssoauthapp.ErrSSOPayloadInvalid
	}
	if s.authUnitOfWork == nil {
		return nil, ssoauthapp.ErrSSOAuthConfigInvalid
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
				user, created, txErr = s.findOrCreateSSOUser(tx, verified)
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
				applySSOIdentity(verified, identity)
				identity.UpdatedAt = time.Now()
				if txErr = tx.CreateIdentity(identity); txErr != nil {
					return txErr
				}
			}
			if applySSOIdentity(verified, identity) {
				identity.UpdatedAt = time.Now()
				if txErr = tx.UpdateIdentity(identity); txErr != nil {
					return txErr
				}
			}
			fields := map[string]interface{}{}
			if s.ssoAuthService.OnlyEnabled() {
				user.DisplayName = resolveSSODisplayName(verified, user.Email)
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

// findOrCreateSSOUser 按邮箱找已有用户；不存在则直接建号。
// SSO 是本部署唯一的注册入口：IdP（自有 Casdoor）注册时已强制邮箱验证码，
// 因此这里有意不检查站点注册开关——关闭本地注册不影响经 IdP 的新用户建号。
// 只有已验证邮箱可以首次关联；不同 subject 不得覆盖已有同提供方绑定。
func (s *Service) findOrCreateSSOUser(tx AuthTransaction, verified *ssoauthapp.IdentityVerified) (*userdomain.User, bool, error) {
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
		DisplayName:           resolveSSODisplayName(verified, email),
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

func resolveSSODisplayName(verified *ssoauthapp.IdentityVerified, email string) string {
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

func applySSOIdentity(verified *ssoauthapp.IdentityVerified, identity *externalidentitydomain.Identity) bool {
	if verified == nil || identity == nil {
		return false
	}
	changed := false
	if identity.Username != verified.Username {
		identity.Username = verified.Username
		changed = true
	}
	avatar, present := ssoAvatar(verified)
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

func ssoAvatar(verified *ssoauthapp.IdentityVerified) (string, bool) {
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
