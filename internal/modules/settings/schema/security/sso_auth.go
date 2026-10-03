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

var ErrSSOAuthConfigInvalid = errors.New("sso auth config invalid")

// SSOAuthSetting configures Casdoor session authentication independently of OAuth.
type SSOAuthSetting struct {
	Enabled       bool   `json:"enabled"`
	OnlyEnabled   bool   `json:"only_enabled"`
	Issuer        string `json:"issuer"`
	ApplicationID string `json:"application_id"`
	Organization  string `json:"organization"`
	DisplayName   string `json:"display_name"`
}

type SSOAuthSettingPatch struct {
	Enabled       *bool   `json:"enabled"`
	OnlyEnabled   *bool   `json:"only_enabled"`
	Issuer        *string `json:"issuer"`
	ApplicationID *string `json:"application_id"`
	Organization  *string `json:"organization"`
	DisplayName   *string `json:"display_name"`
}

func DefaultSSOAuthSetting(cfg config.SSOAuthConfig) SSOAuthSetting {
	return NormalizeSSOAuthSetting(SSOAuthSetting{
		Enabled: cfg.Enabled, OnlyEnabled: cfg.OnlyEnabled, Issuer: cfg.Issuer,
		ApplicationID: cfg.ApplicationID, Organization: cfg.Organization, DisplayName: cfg.DisplayName,
	})
}

func NormalizeSSOAuthSetting(setting SSOAuthSetting) SSOAuthSetting {
	setting.Issuer = strings.TrimRight(strings.TrimSpace(setting.Issuer), "/")
	setting.ApplicationID = strings.TrimSpace(setting.ApplicationID)
	setting.Organization = strings.TrimSpace(setting.Organization)
	setting.DisplayName = strings.TrimSpace(setting.DisplayName)
	return setting
}

func SSOEndpointsReady(setting SSOAuthSetting) bool {
	setting = NormalizeSSOAuthSetting(setting)
	parts := strings.Split(setting.ApplicationID, "/")
	return SSOIdentityPageURL(setting.Issuer, "/account") != "" && setting.Organization != "" &&
		len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != ""
}

func ValidateSSOAuthSetting(setting SSOAuthSetting) error {
	setting = NormalizeSSOAuthSetting(setting)
	if setting.OnlyEnabled && !setting.Enabled {
		return fmt.Errorf("%w: 仅统一登录需要启用通行证认证", ErrSSOAuthConfigInvalid)
	}
	if setting.Enabled && !SSOEndpointsReady(setting) {
		return fmt.Errorf("%w: 通行证认证需要有效 HTTPS 根地址、组织及应用 ID (owner/name)", ErrSSOAuthConfigInvalid)
	}
	return nil
}

func ApplySSOAuthSettingPatch(current SSOAuthSetting, patch SSOAuthSettingPatch) SSOAuthSetting {
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.OnlyEnabled != nil {
		current.OnlyEnabled = *patch.OnlyEnabled
	}
	if patch.Issuer != nil {
		current.Issuer = *patch.Issuer
	}
	if patch.ApplicationID != nil {
		current.ApplicationID = *patch.ApplicationID
	}
	if patch.Organization != nil {
		current.Organization = *patch.Organization
	}
	if patch.DisplayName != nil {
		current.DisplayName = *patch.DisplayName
	}
	return NormalizeSSOAuthSetting(current)
}

func SSOAuthSettingToConfig(setting SSOAuthSetting) config.SSOAuthConfig {
	setting = NormalizeSSOAuthSetting(setting)
	return config.SSOAuthConfig{
		Enabled: setting.Enabled, OnlyEnabled: setting.OnlyEnabled, Issuer: setting.Issuer,
		ApplicationID: setting.ApplicationID, Organization: setting.Organization, DisplayName: setting.DisplayName,
	}
}

func EncodeSSOAuthSetting(setting SSOAuthSetting) jsonmap.JSON {
	setting = NormalizeSSOAuthSetting(setting)
	return jsonmap.JSON{
		"enabled": setting.Enabled, "only_enabled": setting.OnlyEnabled, "issuer": setting.Issuer,
		"application_id": setting.ApplicationID, "organization": setting.Organization, "display_name": setting.DisplayName,
	}
}

func MaskSSOAuthSettingForAdmin(setting SSOAuthSetting) jsonmap.JSON {
	result := EncodeSSOAuthSetting(setting)
	result["admin_url"] = SSOIdentityPageURL(setting.Issuer, "/login/built-in")
	return result
}

// SSOIdentityPageURL accepts only a root HTTPS origin (loopback HTTP is allowed for development).
func SSOIdentityPageURL(issuer, path string) string {
	u, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return ""
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")) {
		return ""
	}
	u.Path, u.RawPath = path, ""
	return u.String()
}

// DecodeSSOAuthSetting preserves explicitly stored false and empty values.
func DecodeSSOAuthSetting(raw jsonmap.JSON, fallback SSOAuthSetting) SSOAuthSetting {
	if value, exists := raw["enabled"]; exists {
		fallback.Enabled = settingsvalue.ParseBool(value)
	}
	if value, exists := raw["only_enabled"]; exists {
		fallback.OnlyEnabled = settingsvalue.ParseBool(value)
	}
	for key, target := range map[string]*string{
		"issuer": &fallback.Issuer, "application_id": &fallback.ApplicationID,
		"organization": &fallback.Organization, "display_name": &fallback.DisplayName,
	} {
		if value, exists := raw[key]; exists {
			if text, ok := value.(string); ok {
				*target = text
			}
		}
	}
	return NormalizeSSOAuthSetting(fallback)
}

func NormalizeSSOAuthSettingJSON(raw jsonmap.JSON) jsonmap.JSON {
	return EncodeSSOAuthSetting(DecodeSSOAuthSetting(raw, SSOAuthSetting{}))
}
