package migrations

import (
	"errors"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	settingsstore "github.com/dujiao-next/internal/modules/settings/infrastructure/gormstore"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ssoAuthMigrationKey = "migration/sso_auth_v1"
const legacyOIDCAuthSettingKey = "oidc_auth_config"

// EnsureSSOAuthSetting imports legacy settings once, before authentication is wired.
// Existing new settings (including explicit false and empty values) are authoritative.
func EnsureSSOAuthSetting(db *gorm.DB, fallback config.SSOAuthConfig) error {
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var marker settingsstore.SettingRecord
		markerErr := tx.First(&marker, "key = ?", ssoAuthMigrationKey).Error
		if markerErr != nil && !errors.Is(markerErr, gorm.ErrRecordNotFound) {
			return markerErr
		}
		var current settingsstore.SettingRecord
		currentErr := tx.First(&current, "key = ?", constants.SettingKeySSOAuthConfig).Error
		if currentErr != nil && !errors.Is(currentErr, gorm.ErrRecordNotFound) {
			return currentErr
		}
		if errors.Is(currentErr, gorm.ErrRecordNotFound) {
			if markerErr == nil && migrationDone(marker.ValueJSON) {
				return errors.New("sso authentication settings are missing after migration")
			}
			setting := settingssecurity.DefaultSSOAuthSetting(fallback)
			var legacy settingsstore.SettingRecord
			legacyErr := tx.First(&legacy, "key = ?", legacyOIDCAuthSettingKey).Error
			if legacyErr != nil && !errors.Is(legacyErr, gorm.ErrRecordNotFound) {
				return legacyErr
			}
			if legacyErr == nil {
				setting = settingssecurity.DecodeSSOAuthSetting(legacy.ValueJSON, setting)
			}
			if err := settingssecurity.ValidateSSOAuthSetting(setting); err != nil {
				return err
			}
			current = settingsstore.SettingRecord{Key: constants.SettingKeySSOAuthConfig, ValueJSON: settingssecurity.EncodeSSOAuthSetting(setting)}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&current).Error; err != nil {
				return err
			}
		}
		marker = settingsstore.SettingRecord{Key: ssoAuthMigrationKey, ValueJSON: jsonmap.JSON{"done": true}}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&marker).Error
	})
}
