package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUnifiedAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	blocked := []string{
		"POST /auth/login", "POST /auth/register", "POST /auth/send-verify-code", "POST /auth/forgot-password",
		"POST /auth/google/login", "POST /auth/google/redirect/intent", "POST /auth/google/redirect/callback", "POST /auth/google/redirect/exchange",
		"POST /auth/telegram/login", "POST /auth/telegram/miniapp/login", "GET /auth/telegram/oidc/start", "POST /auth/telegram/oidc/callback",
		"PUT /me/password", "POST /me/email/send-verify-code", "POST /me/email/change",
		"POST /me/2fa/setup", "POST /me/2fa/enable", "POST /me/2fa/disable", "POST /me/2fa/recovery-codes/regenerate",
		"POST /me/google/bind", "POST /me/google/redirect/intent", "POST /me/google/redirect/exchange", "DELETE /me/google/unbind",
		"POST /me/telegram/bind", "POST /me/telegram/miniapp/bind", "DELETE /me/telegram/unbind", "GET /me/telegram/oidc/start", "POST /me/telegram/oidc/callback",
		"GET /me/oidc/start", "POST /me/oidc/callback", "DELETE /me/oidc/unbind",
	}
	allowed := []string{
		"GET /auth/oidc/start", "POST /auth/oidc/callback", "POST /auth/oidc/password-login", "POST /auth/oidc/mfa", "POST /auth/oidc/register/send-code", "POST /auth/oidc/register",
		"POST /auth/login/verify-2fa", "GET /me/2fa/status", "GET /me/google", "GET /me/telegram", "GET /me/oidc",
		"GET /me", "PUT /me/profile", "POST /orders", "GET /wallet", "GET /orders/:id",
	}
	policy := false
	router := gin.New()
	group := router.Group("/api/v1", UnifiedAuthMiddleware(func() bool { return policy }))
	for _, route := range append(append([]string{}, blocked...), allowed...) {
		parts := strings.SplitN(route, " ", 2)
		group.Handle(parts[0], parts[1], func(c *gin.Context) { c.Status(http.StatusNoContent) })
	}
	// The actual admin group has no customer policy middleware.
	router.POST("/api/v1/admin/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, enabled := range []bool{false, true, false} {
		policy = enabled
		for _, routes := range []struct {
			routes  []string
			blocked bool
		}{{blocked, true}, {allowed, false}, {[]string{"POST /admin/login"}, false}} {
			for _, route := range routes.routes {
				parts := strings.SplitN(route, " ", 2)
				request := httptest.NewRequest(parts[0], "/api/v1"+strings.ReplaceAll(parts[1], ":id", "1")+"?local=1", nil)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				want := http.StatusNoContent
				if enabled && routes.blocked {
					want = http.StatusForbidden
				}
				if response.Code != want {
					t.Errorf("policy=%v %s = %d, want %d", enabled, route, response.Code, want)
				}
			}
		}
	}
}

func TestUnifiedAuthMiddlewareRequiresPolicy(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil dependency panic")
		}
	}()
	UnifiedAuthMiddleware(nil)
}
