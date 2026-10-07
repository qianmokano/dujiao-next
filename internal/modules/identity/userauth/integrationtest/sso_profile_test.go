package integrationtest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	ssoauthapp "github.com/dujiao-next/internal/modules/identity/ssoauth/application"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	userauthapp "github.com/dujiao-next/internal/modules/identity/userauth/application"
	userauthstore "github.com/dujiao-next/internal/modules/identity/userauth/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func oidcProfileFixture(t *testing.T, account map[string]interface{}) (*userauthapp.Service, *gorm.DB, *userdomain.User) {
	t.Helper()
	if _, exists := account["email"]; !exists {
		account["email"] = "original@example.com"
		account["emailVerified"] = true
	}
	svc, _, db := setupTelegramOAuthTestService(t)
	now := time.Now()
	user := &userdomain.User{Email: "original@example.com", PasswordHash: "untouched", DisplayName: "Original", Status: constants.UserStatusActive, MemberLevelID: 7, TotalSpent: money.FromDecimal(decimal.NewFromInt(42)), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&externalidentitydomain.Identity{UserID: user.ID, Provider: constants.UserOAuthProviderOIDC, ProviderUserID: "subject", Username: "old", AvatarURL: "https://auth.example/old.png", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login":
			http.SetCookie(w, &http.Cookie{Name: "casdoor_session_id", Value: "session"})
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "data": "kano/buyer"})
		case "/api/get-account":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "data": account})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	svc.SetSSOAuthService(ssoauthapp.NewService(config.SSOAuthConfig{Enabled: true, OnlyEnabled: true, Issuer: server.URL, ApplicationID: "admin/store", Organization: "kano"}))
	return svc, db, user
}

func profileLogin(svc *userauthapp.Service) (*userauthapp.SSOPasswordLoginResult, error) {
	return svc.LoginWithSSOPassword(userauthapp.SSOPasswordLoginInput{Account: "buyer", Password: "password", Context: context.Background()})
}

func TestOIDCProfileSynchronizesOnRepeatedLogin(t *testing.T) {
	for _, tc := range []struct {
		name, displayName, username, wantName, wantAvatar string
		avatar                                            interface{}
		present                                           bool
	}{
		{"display name and avatar", "Passport", "buyer", "Passport", "https://auth.example/new.png", "https://auth.example/new.png", true},
		{"username fallback", " ", "buyer", "buyer", "https://auth.example/old.png", nil, false},
		{"original email fallback", "", "", "original", "https://auth.example/old.png", nil, false},
		{"explicit clear", "Passport", "buyer", "Passport", "", "", true},
		{"invalid URL preserved", "Passport", "buyer", "Passport", "https://auth.example/old.png", "javascript:alert(1)", true},
		{"null preserved", "Passport", "buyer", "Passport", "https://auth.example/old.png", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := map[string]interface{}{"id": "subject", "name": tc.username, "displayName": tc.displayName, "email": "changed@example.com", "emailVerified": true}
			if tc.present {
				account["avatar"] = tc.avatar
			}
			svc, db, original := oidcProfileFixture(t, account)
			for repeat := 0; repeat < 2; repeat++ {
				result, err := profileLogin(svc)
				if err != nil || result == nil || result.Login == nil || result.Login.Token == "" {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				if result.Login.User.ID != original.ID || result.Login.User.DisplayName != tc.wantName {
					t.Fatalf("user=%+v", result.Login.User)
				}
			}
			var stored userdomain.User
			if err := db.First(&stored, original.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Email != original.Email || stored.PasswordHash != original.PasswordHash || stored.MemberLevelID != original.MemberLevelID || !stored.TotalSpent.Decimal.Equal(original.TotalSpent.Decimal) || stored.EmailVerifiedAt != nil {
				t.Fatalf("business or identity source changed: %+v", stored)
			}
			avatar, err := svc.GetProfileAvatar(original.ID)
			if err != nil || avatar != tc.wantAvatar {
				t.Fatalf("avatar=%q err=%v", avatar, err)
			}
			var count int64
			db.Model(&userdomain.User{}).Count(&count)
			if count != 1 {
				t.Fatalf("duplicate accounts: %d", count)
			}
		})
	}
}

type failingOIDCProfileTransaction struct{ userauthapp.AuthTransaction }

func (t failingOIDCProfileTransaction) UpdateUserFields(userID uint, fields map[string]interface{}) error {
	if err := t.AuthTransaction.UpdateUserFields(userID, fields); err != nil {
		return err
	}
	return errors.New("profile write failed after mutation")
}

type failingOIDCProfileWork struct{ userauthapp.AuthUnitOfWork }

func (u failingOIDCProfileWork) WithinTransaction(ctx context.Context, fn func(userauthapp.AuthTransaction) error) error {
	return u.AuthUnitOfWork.WithinTransaction(ctx, func(tx userauthapp.AuthTransaction) error { return fn(failingOIDCProfileTransaction{tx}) })
}

func TestOIDCProfileFailureRollsBackIdentityAndNickname(t *testing.T) {
	svc, db, original := oidcProfileFixture(t, map[string]interface{}{"id": "subject", "name": "buyer", "displayName": "Must roll back", "avatar": "https://auth.example/new.png"})
	svc.SetAuthUnitOfWork(failingOIDCProfileWork{userauthstore.New(db)})
	result, err := profileLogin(svc)
	if err == nil || result != nil {
		t.Fatalf("failed sync signed in: %+v %v", result, err)
	}
	var stored userdomain.User
	if err := db.First(&stored, original.ID).Error; err != nil {
		t.Fatal(err)
	}
	var identity externalidentitydomain.Identity
	if err := db.Where("user_id = ?", original.ID).First(&identity).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DisplayName != "Original" || stored.LastLoginAt != nil || identity.Username != "old" || identity.AvatarURL != "https://auth.example/old.png" {
		t.Fatalf("partial update: %+v %+v", stored, identity)
	}
}

type businessUpdateAfterOIDCSync struct {
	userauthapp.AuthUnitOfWork
	db     *gorm.DB
	userID uint
}

func (u businessUpdateAfterOIDCSync) WithinTransaction(ctx context.Context, fn func(userauthapp.AuthTransaction) error) error {
	if err := u.AuthUnitOfWork.WithinTransaction(ctx, fn); err != nil {
		return err
	}
	return u.db.Model(&userdomain.User{}).Where("id = ?", u.userID).Updates(map[string]interface{}{"member_level_id": 9, "total_spent": money.FromDecimal(decimal.NewFromInt(99))}).Error
}

func TestOIDCLoginPreservesInterleavedBusinessUpdatesAndLocale(t *testing.T) {
	svc, db, original := oidcProfileFixture(t, map[string]interface{}{"id": "subject", "name": "buyer", "displayName": "Passport"})
	svc.SetAuthUnitOfWork(businessUpdateAfterOIDCSync{userauthstore.New(db), db, original.ID})
	if err := svc.CheckLocalIdentityManagement(); !errors.Is(err, userauthapp.ErrUnifiedAuthRequired) {
		t.Fatalf("identity editing opened in unified mode: %v", err)
	}
	if _, err := profileLogin(svc); err != nil {
		t.Fatal(err)
	}
	name, locale := "Local", "en-US"
	if _, err := svc.UpdateProfile(original.ID, &name, &locale); !errors.Is(err, userauthapp.ErrUnifiedAuthRequired) {
		t.Fatalf("nickname update: %v", err)
	}
	if _, err := svc.UpdateProfile(original.ID, nil, &locale); err != nil {
		t.Fatal(err)
	}
	var stored userdomain.User
	if err := db.First(&stored, original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.MemberLevelID != 9 || !stored.TotalSpent.Decimal.Equal(decimal.NewFromInt(99)) || stored.DisplayName != "Passport" || stored.Locale != locale {
		t.Fatalf("interleaved changes lost: %+v", stored)
	}
}
