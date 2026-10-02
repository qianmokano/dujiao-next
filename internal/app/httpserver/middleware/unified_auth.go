package middleware

import (
	"net/http"
	"strings"

	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

// UnifiedAuthMiddleware restricts customer authentication and identity changes.
// Attach it only to storefront auth and authenticated customer route groups;
// independent admin and guest commerce routes keep their existing policies.
func UnifiedAuthMiddleware(onlyEnabled func() bool) gin.HandlerFunc {
	if onlyEnabled == nil {
		panic("unified auth middleware: policy is nil")
	}
	return func(c *gin.Context) {
		if !onlyEnabled() || unifiedAuthRouteAllowed(c.Request.Method, c.FullPath()) {
			c.Next()
			return
		}
		response.ErrorWithHTTPStatus(c, http.StatusForbidden, response.CodeForbidden, i18n.T(i18n.ResolveLocale(c), "error.unified_auth_required"))
	}
}

func unifiedAuthRouteAllowed(method, path string) bool {
	if index := strings.Index(path, "/auth/"); index >= 0 {
		endpoint := path[index+len("/auth/"):]
		if method == http.MethodGet {
			return endpoint == "oidc/start"
		}
		if method == http.MethodPost {
			switch endpoint {
			case "oidc/callback", "oidc/password-login", "oidc/mfa", "oidc/register/send-code", "oidc/register", "login/verify-2fa":
				return true
			}
		}
		return false
	}
	if index := strings.Index(path, "/me/"); index >= 0 {
		endpoint := path[index+len("/me/"):]
		for _, prefix := range []string{"password", "email/", "2fa/", "google", "telegram", "oidc"} {
			if strings.HasPrefix(endpoint, prefix) {
				return method == http.MethodGet && (endpoint == "2fa/status" || endpoint == "google" || endpoint == "telegram" || endpoint == "oidc")
			}
		}
	}
	return true
}
