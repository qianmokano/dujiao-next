package settingssecurity

import (
	"errors"
	"testing"

	"github.com/dujiao-next/internal/config"
)

func TestOIDCOnlyPolicyRoundTripAndValidation(t *testing.T) {
	cfg := config.OIDCAuthConfig{Enabled: true, OnlyEnabled: true, Issuer: " https://auth.example.com/ ", ClientID: "client", ClientSecret: "secret", RedirectURI: "https://store.example.com/callback", ApplicationID: "admin/store", Organization: "kano"}
	setting := DefaultOIDCAuthSetting(cfg)
	if err := ValidateOIDCAuthSetting(setting); err != nil {
		t.Fatal(err)
	}
	decoded := DecodeOIDCAuthSetting(EncodeOIDCAuthSetting(setting), OIDCAuthSetting{})
	if !decoded.OnlyEnabled || !OIDCAuthSettingToConfig(decoded).OnlyEnabled || MaskOIDCAuthSettingForAdmin(decoded)["only_enabled"] != true {
		t.Fatal("policy lost in round trip")
	}
	if MaskOIDCAuthSettingForAdmin(decoded)["client_secret"] != "" {
		t.Fatal("secret exposed")
	}
	disabled := false
	next := ApplyOIDCAuthSettingPatch(decoded, OIDCAuthSettingPatch{Enabled: &disabled})
	if !errors.Is(ValidateOIDCAuthSetting(next), ErrOIDCAuthConfigInvalid) {
		t.Fatal("disabled provider must reject only policy")
	}
	next = ApplyOIDCAuthSettingPatch(next, OIDCAuthSettingPatch{OnlyEnabled: &disabled})
	if err := ValidateOIDCAuthSetting(next); err != nil {
		t.Fatal(err)
	}
	legacy := DecodeOIDCAuthSetting(map[string]interface{}{"enabled": false}, OIDCAuthSetting{})
	if legacy.OnlyEnabled {
		t.Fatal("legacy policy must remain optional")
	}
}
