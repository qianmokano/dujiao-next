package oidcauthapp

import (
	"sync"
	"testing"

	"github.com/dujiao-next/internal/config"
)

func TestUnifiedPublicPolicy(t *testing.T) {
	var nilService *Service
	if nilService.OnlyEnabled() || nilService.PublicConfig()["enabled"] != false {
		t.Fatal("nil service must not advertise login")
	}
	cfg := config.OIDCAuthConfig{Enabled: true, OnlyEnabled: true, Issuer: "https://auth.example.com/", ClientID: "client", ClientSecret: "secret", RedirectURI: "https://store.example.com/callback", ApplicationID: "admin/store", Organization: "kano"}
	svc := NewService(cfg)
	public := svc.PublicConfig()
	if public["only_enabled"] != true || public["enabled"] != true || public["account_url"] != "https://auth.example.com/login/kano" || public["password_reset_url"] != "https://auth.example.com/forget/store" {
		t.Fatalf("public=%v", public)
	}
	if _, exists := public["client_secret"]; exists {
		t.Fatal("public config exposed secret")
	}
	cfg.Enabled = false
	svc.SetConfig(cfg)
	if !svc.OnlyEnabled() || svc.PublicConfig()["enabled"] != false {
		t.Fatal("disabled provider must not reopen local auth")
	}
	cfg.Issuer = "javascript:alert(1)"
	svc.SetConfig(cfg)
	if svc.PublicConfig()["account_url"] != "" {
		t.Fatal("unsafe identity URL")
	}
	if identityPageURL(config.OIDCAuthConfig{Issuer: "https://auth.example.com"}, "login", "") != "" {
		t.Fatal("missing identity page")
	}
}

func TestUnifiedPolicyConcurrentUpdate(t *testing.T) {
	svc := NewService(config.OIDCAuthConfig{})
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 50; j++ {
				svc.SetConfig(config.OIDCAuthConfig{OnlyEnabled: j%2 == 0})
				_ = svc.OnlyEnabled()
				_ = svc.PublicConfig()
			}
		}()
	}
	workers.Wait()
}
