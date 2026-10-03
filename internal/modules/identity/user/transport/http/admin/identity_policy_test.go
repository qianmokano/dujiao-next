package adminuserhttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	"github.com/gin-gonic/gin"
)

func TestUnifiedAdminIdentityWritesAreForbiddenBeforeBusinessMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, field := range []string{"nickname", "email", "password", "avatar", "avatar_url", "email_verified", "email_verified_at", "oauth_identities", "auth_bindings", "auth_identities", "role"} {
		t.Run(field, func(t *testing.T) {
			h := &AdminHandler{identityPolicy: func() (bool, error) { return true, nil }}
			router := gin.New()
			router.PUT("/users/:id", h.UpdateAdminUser)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/users/42", strings.NewReader(`{"`+field+`":null,"admin_note":"must not save","status":"disabled"}`))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("HTTP %d: %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

type identityPolicyUsers struct {
	UserDirectory
	updates map[string]interface{}
}

func (s *identityPolicyUsers) GetByID(uint) (*userdomain.User, error) {
	return &userdomain.User{ID: 42, DisplayName: "Passport", Status: "active", MemberLevelID: 7}, nil
}
func (s *identityPolicyUsers) UpdateFields(_ uint, updates map[string]interface{}) error {
	s.updates = updates
	return nil
}

func TestAdminIdentityPolicyAllowsBusinessAndLocalModeAndClosesOnError(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		only       bool
		policyErr  error
		wantHTTP   int
		wantName   bool
	}{
		{"unified business", `{"admin_note":"business","locale":"en-US"}`, true, nil, 200, false},
		{"local nickname", `{"nickname":"Local"}`, false, nil, 200, true},
		{"policy unavailable", `{"nickname":"Local","admin_note":"must not save"}`, false, errors.New("settings unavailable"), 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := &identityPolicyUsers{}
			h := &AdminHandler{users: users, identityPolicy: func() (bool, error) { return tc.only, tc.policyErr }}
			router := gin.New()
			router.PUT("/users/:id", h.UpdateAdminUser)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/users/42", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.wantHTTP {
				t.Fatalf("HTTP %d", recorder.Code)
			}
			if tc.policyErr != nil {
				if users.updates != nil {
					t.Fatal("failed policy wrote fields")
				}
				return
			}
			_, hasName := users.updates["display_name"]
			if hasName != tc.wantName {
				t.Fatalf("updates=%v", users.updates)
			}
			for _, key := range []string{"total_spent", "total_recharged", "member_level_id", "email", "password_hash"} {
				if _, exists := users.updates[key]; exists {
					t.Fatalf("overwrites unrelated %s", key)
				}
			}
		})
	}
}

func TestUnifiedAdminIdentityUnbindIsForbidden(t *testing.T) {
	for _, unbind := range []func(*AdminHandler) gin.HandlerFunc{
		func(h *AdminHandler) gin.HandlerFunc { return h.UnbindAdminUserGoogle },
		func(h *AdminHandler) gin.HandlerFunc { return h.UnbindAdminUserTelegram },
	} {
		h := &AdminHandler{identityPolicy: func() (bool, error) { return true, nil }}
		router := gin.New()
		router.DELETE("/users/:id", unbind(h))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/users/42", nil))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("HTTP %d", recorder.Code)
		}
	}
}
