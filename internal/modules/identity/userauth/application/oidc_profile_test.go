package application

import (
	oidcauthapp "github.com/dujiao-next/internal/modules/identity/oidcauth/application"
	"testing"
)

func TestOIDCProfileNameAndAvatarRules(t *testing.T) {
	for _, tc := range []struct{ display, username, email, want string }{
		{" Passport ", "user", "original@example.com", "Passport"},
		{" ", " user ", "original@example.com", "user"},
		{"", "", "original@example.com", "original"},
		{"", "", "original", "original"},
	} {
		if got := resolveOIDCDisplayName(&oidcauthapp.IdentityVerified{DisplayName: tc.display, Username: tc.username}, tc.email); got != tc.want {
			t.Fatalf("name=%q want=%q", got, tc.want)
		}
	}
	for _, tc := range []struct {
		value                string
		present, wantPresent bool
		want                 string
	}{
		{"", false, false, ""}, {" ", true, true, ""},
		{" https://auth.example/avatar.png ", true, true, "https://auth.example/avatar.png"},
		{"http://auth.example/avatar.png", false, true, "http://auth.example/avatar.png"},
		{"javascript:alert(1)", true, false, ""}, {"https://user:secret@auth.example/avatar", true, false, ""},
		{"/avatar.png", true, false, ""}, {":invalid", true, false, ""},
	} {
		got, present := oidcAvatar(&oidcauthapp.IdentityVerified{AvatarURL: tc.value, AvatarPresent: tc.present})
		if got != tc.want || present != tc.wantPresent {
			t.Fatalf("avatar=%q present=%v for %+v", got, present, tc)
		}
	}
}
