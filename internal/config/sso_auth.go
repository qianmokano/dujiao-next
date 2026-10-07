package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

var ssoAuthFields = []string{"enabled", "only_enabled", "issuer", "application_id", "organization", "display_name"}

func ssoAuthConfigPresent(v *viper.Viper, root string) bool {
	if v.InConfig(root) {
		return true
	}
	for _, field := range ssoAuthFields {
		if _, exists := os.LookupEnv(strings.ToUpper(root + "_" + field)); exists {
			return true
		}
	}
	return false
}

func ssoAuthConfigFromViper(v *viper.Viper, root string) SSOAuthConfig {
	return SSOAuthConfig{
		Enabled: v.GetBool(root + ".enabled"), OnlyEnabled: v.GetBool(root + ".only_enabled"),
		Issuer: v.GetString(root + ".issuer"), ApplicationID: v.GetString(root + ".application_id"),
		Organization: v.GetString(root + ".organization"), DisplayName: v.GetString(root + ".display_name"),
	}
}

// loadSSOAuthMigrationInputs reads only the six session API fields from legacy YAML/environment.
func loadSSOAuthMigrationInputs(v *viper.Viper, cfg *Config) {
	cfg.SSOAuthConfigured = ssoAuthConfigPresent(v, "sso_auth")
	if cfg.SSOAuthConfigured {
		cfg.SSOAuth = ssoAuthConfigFromViper(v, "sso_auth")
	}
	if ssoAuthConfigPresent(v, "oidc_auth") {
		legacy := ssoAuthConfigFromViper(v, "oidc_auth")
		cfg.LegacySSOAuth = &legacy
	}
}
