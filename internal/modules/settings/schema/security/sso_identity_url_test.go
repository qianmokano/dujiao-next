package settingssecurity

import "testing"

func TestSSOIdentityPageURLAndReadOnlyAdminLink(t *testing.T) {
	for _, tc := range []struct{ issuer, want string }{
		{"https://auth.example", "https://auth.example/login/built-in"},
		{"https://auth.example/", "https://auth.example/login/built-in"},
		{"http://127.0.0.1:8000", "http://127.0.0.1:8000/login/built-in"},
		{"http://auth.example", ""}, {"https://user:secret@auth.example", ""},
		{"https://auth.example/path", ""}, {"https://auth.example?redirect=evil", ""},
		{"https://auth.example#evil", ""}, {"javascript:alert(1)", ""}, {"", ""},
	} {
		setting := SSOAuthSetting{Issuer: tc.issuer}
		if got := MaskSSOAuthSettingForAdmin(setting)["admin_url"]; got != tc.want {
			t.Fatalf("issuer=%q got=%v want=%s", tc.issuer, got, tc.want)
		}
		if _, persisted := EncodeSSOAuthSetting(setting)["admin_url"]; persisted {
			t.Fatal("computed link was persisted")
		}
	}
}
