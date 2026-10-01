package settingssecurity

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/dujiao-next/internal/config"
	settingsvalue "github.com/dujiao-next/internal/modules/settings/schema/value"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

var ErrOIDCAuthConfigInvalid = errors.New("oidc auth config invalid")

// OIDCAuthSetting 通用 OIDC（单点登录）配置实体。
type OIDCAuthSetting struct {
	Enabled      bool   `json:"enabled"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	DisplayName  string `json:"display_name"`
}

// OIDCAuthSettingPatch 通用 OIDC 配置补丁。
type OIDCAuthSettingPatch struct {
	Enabled      *bool   `json:"enabled"`
	Issuer       *string `json:"issuer"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	RedirectURI  *string `json:"redirect_uri"`
	DisplayName  *string `json:"display_name"`
}

// DefaultOIDCAuthSetting 根据运行时配置生成默认设置。
func DefaultOIDCAuthSetting(cfg config.OIDCAuthConfig) OIDCAuthSetting {
	return NormalizeOIDCAuthSetting(OIDCAuthSetting{
		Enabled:      cfg.Enabled,
		Issuer:       strings.TrimSpace(cfg.Issuer),
		ClientID:     strings.TrimSpace(cfg.ClientID),
		ClientSecret: strings.TrimSpace(cfg.ClientSecret),
		RedirectURI:  strings.TrimSpace(cfg.RedirectURI),
		DisplayName:  strings.TrimSpace(cfg.DisplayName),
	})
}

// NormalizeOIDCAuthSetting 归一化通用 OIDC 配置。
func NormalizeOIDCAuthSetting(setting OIDCAuthSetting) OIDCAuthSetting {
	setting.Issuer = strings.TrimRight(strings.TrimSpace(setting.Issuer), "/")
	setting.ClientID = strings.TrimSpace(setting.ClientID)
	setting.ClientSecret = strings.TrimSpace(setting.ClientSecret)
	setting.RedirectURI = strings.TrimRight(strings.TrimSpace(setting.RedirectURI), "/")
	setting.DisplayName = strings.TrimSpace(setting.DisplayName)
	return setting
}

// OIDCEndpointsReady 判断 OIDC 端点配置是否齐备（不含 Enabled）。
func OIDCEndpointsReady(setting OIDCAuthSetting) bool {
	normalized := NormalizeOIDCAuthSetting(setting)
	return normalized.Issuer != "" &&
		normalized.ClientID != "" &&
		normalized.ClientSecret != "" &&
		normalized.RedirectURI != ""
}

// ValidateOIDCAuthSetting 校验通用 OIDC 配置合法性。
func ValidateOIDCAuthSetting(setting OIDCAuthSetting) error {
	normalized := NormalizeOIDCAuthSetting(setting)
	if !normalized.Enabled {
		return nil
	}
	if err := validateOIDCHttpsURL(normalized.Issuer); err != nil {
		return fmt.Errorf("%w: Issuer %v", ErrOIDCAuthConfigInvalid, err)
	}
	if normalized.ClientID == "" {
		return fmt.Errorf("%w: Client ID 不能为空", ErrOIDCAuthConfigInvalid)
	}
	if normalized.ClientSecret == "" {
		return fmt.Errorf("%w: Client Secret 不能为空", ErrOIDCAuthConfigInvalid)
	}
	if err := validateOIDCHttpsURL(normalized.RedirectURI); err != nil {
		return fmt.Errorf("%w: 回调地址 %v", ErrOIDCAuthConfigInvalid, err)
	}
	return nil
}

func validateOIDCHttpsURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("不能为空")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("必须是合法的 http(s) URL")
	}
	return nil
}

// ApplyOIDCAuthSettingPatch 把补丁应用到当前配置（不做校验）。
func ApplyOIDCAuthSettingPatch(current OIDCAuthSetting, patch OIDCAuthSettingPatch) OIDCAuthSetting {
	next := current
	if patch.Enabled != nil {
		next.Enabled = *patch.Enabled
	}
	if patch.Issuer != nil {
		next.Issuer = strings.TrimSpace(*patch.Issuer)
	}
	if patch.ClientID != nil {
		next.ClientID = strings.TrimSpace(*patch.ClientID)
	}
	if patch.ClientSecret != nil {
		if v := strings.TrimSpace(*patch.ClientSecret); v != "" {
			next.ClientSecret = v
		}
	}
	if patch.RedirectURI != nil {
		next.RedirectURI = strings.TrimSpace(*patch.RedirectURI)
	}
	if patch.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*patch.DisplayName)
	}
	return next
}

// OIDCAuthSettingToConfig 转换为运行时配置。
func OIDCAuthSettingToConfig(setting OIDCAuthSetting) config.OIDCAuthConfig {
	normalized := NormalizeOIDCAuthSetting(setting)
	return config.OIDCAuthConfig{
		Enabled:      normalized.Enabled,
		Issuer:       normalized.Issuer,
		ClientID:     normalized.ClientID,
		ClientSecret: normalized.ClientSecret,
		RedirectURI:  normalized.RedirectURI,
		DisplayName:  normalized.DisplayName,
	}
}

// EncodeOIDCAuthSetting 转换为 settings 存储结构。
func EncodeOIDCAuthSetting(setting OIDCAuthSetting) jsonmap.JSON {
	normalized := NormalizeOIDCAuthSetting(setting)
	return jsonmap.JSON{
		"enabled":       normalized.Enabled,
		"issuer":        normalized.Issuer,
		"client_id":     normalized.ClientID,
		"client_secret": normalized.ClientSecret,
		"redirect_uri":  normalized.RedirectURI,
		"display_name":  normalized.DisplayName,
	}
}

// MaskOIDCAuthSettingForAdmin 返回脱敏配置。
func MaskOIDCAuthSettingForAdmin(setting OIDCAuthSetting) jsonmap.JSON {
	normalized := NormalizeOIDCAuthSetting(setting)
	return jsonmap.JSON{
		"enabled":           normalized.Enabled,
		"issuer":            normalized.Issuer,
		"client_id":         normalized.ClientID,
		"client_secret":     "",
		"has_client_secret": normalized.ClientSecret != "",
		"redirect_uri":      normalized.RedirectURI,
		"display_name":      normalized.DisplayName,
	}
}

// DecodeOIDCAuthSetting 从持久化 JSON 解码，并对缺失字段使用 fallback。
func DecodeOIDCAuthSetting(raw jsonmap.JSON, fallback OIDCAuthSetting) OIDCAuthSetting {
	next := fallback
	if raw == nil {
		return next
	}
	if value, exists := raw["enabled"]; exists {
		next.Enabled = settingsvalue.ParseBool(value)
	}
	for key, target := range map[string]*string{
		"issuer":        &next.Issuer,
		"client_id":     &next.ClientID,
		"client_secret": &next.ClientSecret,
		"redirect_uri":  &next.RedirectURI,
		"display_name":  &next.DisplayName,
	} {
		if value, exists := raw[key]; exists {
			if text, ok := value.(string); ok {
				*target = text
			}
		}
	}
	return next
}

// NormalizeOIDCAuthSettingJSON 是 Registry 使用的原始 JSON 写入策略。
func NormalizeOIDCAuthSettingJSON(raw jsonmap.JSON) jsonmap.JSON {
	return EncodeOIDCAuthSetting(DecodeOIDCAuthSetting(raw, DefaultOIDCAuthSetting(config.OIDCAuthConfig{})))
}
