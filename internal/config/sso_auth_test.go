package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestSSOAuthMigrationInputs(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("oidc_auth:\n  enabled: true\n  only_enabled: true\n  issuer: https://legacy.example\n  client_secret: never-import\nsso_auth:\n  enabled: false\n  issuer: ''\n")); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	loadSSOAuthMigrationInputs(v, &cfg)
	if !cfg.SSOAuthConfigured || cfg.SSOAuth.Enabled || cfg.SSOAuth.Issuer != "" || cfg.LegacySSOAuth == nil || !cfg.LegacySSOAuth.OnlyEnabled || cfg.LegacySSOAuth.Issuer != "https://legacy.example" {
		t.Fatalf("migration precedence lost: %+v", cfg.SSOAuth)
	}
}

func TestSSOAuthEnvironmentPresence(t *testing.T) {
	t.Setenv("SSO_AUTH_ENABLED", "false")
	t.Setenv("SSO_AUTH_ISSUER", "")
	t.Setenv("OIDC_AUTH_ENABLED", "true")
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	var cfg Config
	loadSSOAuthMigrationInputs(v, &cfg)
	if !cfg.SSOAuthConfigured || cfg.SSOAuth.Enabled || cfg.LegacySSOAuth == nil || !cfg.LegacySSOAuth.Enabled {
		t.Fatal("explicit environment policy was lost")
	}
}
