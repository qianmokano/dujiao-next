package settingssecurity

import (
	"errors"
	"testing"

	"github.com/dujiao-next/internal/config"
)

func TestOIDCOnlyPolicyRoundTripAndValidation(t *testing.T) {
	cfg := config.SSOAuthConfig{Enabled: true, OnlyEnabled: true, Issuer: " https://auth.example.com/ ", ApplicationID: "admin/store", Organization: "kano"}
	setting := DefaultSSOAuthSetting(cfg)
	if err := ValidateSSOAuthSetting(setting); err != nil {
		t.Fatal(err)
	}
	decoded := DecodeSSOAuthSetting(EncodeSSOAuthSetting(setting), SSOAuthSetting{})
	if !decoded.OnlyEnabled || !SSOAuthSettingToConfig(decoded).OnlyEnabled || MaskSSOAuthSettingForAdmin(decoded)["only_enabled"] != true {
		t.Fatal("policy lost in round trip")
	}
	if _, exists := MaskSSOAuthSettingForAdmin(decoded)["client_secret"]; exists {
		t.Fatal("secret exposed")
	}
	disabled := false
	next := ApplySSOAuthSettingPatch(decoded, SSOAuthSettingPatch{Enabled: &disabled})
	if !errors.Is(ValidateSSOAuthSetting(next), ErrSSOAuthConfigInvalid) {
		t.Fatal("disabled provider must reject only policy")
	}
	next = ApplySSOAuthSettingPatch(next, SSOAuthSettingPatch{OnlyEnabled: &disabled})
	if err := ValidateSSOAuthSetting(next); err != nil {
		t.Fatal(err)
	}
	legacy := DecodeSSOAuthSetting(map[string]interface{}{"enabled": false}, SSOAuthSetting{})
	if legacy.OnlyEnabled {
		t.Fatal("legacy policy must remain optional")
	}
}
