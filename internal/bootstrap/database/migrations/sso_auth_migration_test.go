package migrations

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	settingsstore "github.com/dujiao-next/internal/modules/settings/infrastructure/gormstore"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func ssoMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "settings.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&settingsstore.SettingRecord{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestSSOAuthMigrationPrecedenceAndHistory(t *testing.T) {
	fallback := config.SSOAuthConfig{Enabled: true, OnlyEnabled: true, Issuer: "https://runtime.example", ApplicationID: "admin/runtime", Organization: "runtime", DisplayName: "Runtime"}
	for _, tc := range []struct {
		name            string
		legacy, current jsonmap.JSON
		want            settingssecurity.SSOAuthSetting
	}{
		{name: "new install", want: settingssecurity.DefaultSSOAuthSetting(fallback)},
		{name: "legacy database", legacy: jsonmap.JSON{"issuer": "https://legacy.example", "application_id": "admin/old", "organization": "old", "display_name": "Old"}, want: settingssecurity.SSOAuthSetting{Enabled: true, OnlyEnabled: true, Issuer: "https://legacy.example", ApplicationID: "admin/old", Organization: "old", DisplayName: "Old"}},
		{name: "explicit legacy disable and empty", legacy: jsonmap.JSON{"enabled": false, "only_enabled": false, "issuer": ""}, want: settingssecurity.SSOAuthSetting{Issuer: "", ApplicationID: "admin/runtime", Organization: "runtime", DisplayName: "Runtime"}},
		{name: "new database wins", legacy: jsonmap.JSON{"enabled": true, "only_enabled": true, "issuer": "https://legacy.example"}, current: jsonmap.JSON{"enabled": false, "only_enabled": false, "issuer": ""}, want: settingssecurity.SSOAuthSetting{ApplicationID: "admin/runtime", Organization: "runtime", DisplayName: "Runtime"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := ssoMigrationDB(t)
			for key, value := range map[string]jsonmap.JSON{legacyOIDCAuthSettingKey: tc.legacy, constants.SettingKeySSOAuthConfig: tc.current} {
				if value != nil {
					if err := db.Create(&settingsstore.SettingRecord{Key: key, ValueJSON: value}).Error; err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := EnsureSSOAuthSetting(db, fallback); err != nil {
				t.Fatal(err)
			}
			var record settingsstore.SettingRecord
			if err := db.First(&record, "key = ?", constants.SettingKeySSOAuthConfig).Error; err != nil {
				t.Fatal(err)
			}
			got := settingssecurity.DecodeSSOAuthSetting(record.ValueJSON, settingssecurity.DefaultSSOAuthSetting(fallback))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got=%+v want=%+v", got, tc.want)
			}
			for _, removed := range []string{"client_id", "client_secret", "redirect_uri"} {
				if _, exists := record.ValueJSON[removed]; exists {
					t.Fatalf("legacy protocol field persisted: %s", removed)
				}
			}
			if tc.current != nil && !reflect.DeepEqual(record.ValueJSON, tc.current) {
				t.Fatal("new partial setting rewritten")
			}
			if tc.legacy != nil {
				var old settingsstore.SettingRecord
				if err := db.First(&old, "key = ?", legacyOIDCAuthSettingKey).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(old.ValueJSON, tc.legacy) {
					t.Fatal("history was changed")
				}
			}
			before := record.ValueJSON
			if err := db.Where("key = ?", legacyOIDCAuthSettingKey).Delete(&settingsstore.SettingRecord{}).Error; err != nil {
				t.Fatal(err)
			}
			if err := EnsureSSOAuthSetting(db, config.SSOAuthConfig{}); err != nil {
				t.Fatal(err)
			}
			if err := db.First(&record, "key = ?", constants.SettingKeySSOAuthConfig).Error; err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(record.ValueJSON, before) {
				t.Fatal("second startup rewrote configuration")
			}
			if err := db.Where("key = ?", constants.SettingKeySSOAuthConfig).Delete(&settingsstore.SettingRecord{}).Error; err != nil {
				t.Fatal(err)
			}
			if err := EnsureSSOAuthSetting(db, fallback); err == nil {
				t.Fatal("deleted new settings resurrected from runtime")
			}
		})
	}
}

func TestSSOAuthMigrationRollsBackAndFailsClosed(t *testing.T) {
	if err := EnsureSSOAuthSetting(nil, config.SSOAuthConfig{}); err == nil {
		t.Fatal("nil database accepted")
	}
	db := ssoMigrationDB(t)
	if err := EnsureSSOAuthSetting(db, config.SSOAuthConfig{Enabled: true}); err == nil {
		t.Fatal("invalid enabled configuration accepted")
	}
	var count int64
	if err := db.Model(&settingsstore.SettingRecord{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("migration was not rolled back: count=%d err=%v", count, err)
	}
	if err := db.Migrator().DropTable(&settingsstore.SettingRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSSOAuthSetting(db, config.SSOAuthConfig{}); err == nil {
		t.Fatal("database failure ignored")
	}
}
