package userauthhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	"github.com/gin-gonic/gin"
)

type profilePolicyService struct {
	policyErr error
	updates   int
}

func (*profilePolicyService) GetUserByID(uint) (*userdomain.User, error) {
	return &userdomain.User{ID: 42, DisplayName: "Passport", Email: "original@example.com"}, nil
}
func (*profilePolicyService) ResolveEmailChangeMode(*userdomain.User) (string, error) { return "", nil }
func (*profilePolicyService) ResolvePasswordChangeMode(*userdomain.User) (string, error) {
	return "", nil
}
func (s *profilePolicyService) CheckLocalIdentityManagement() error { return s.policyErr }
func (*profilePolicyService) GetProfileAvatar(uint) (string, error) {
	return "https://auth.example/avatar.png", nil
}
func (s *profilePolicyService) UpdateProfile(_ uint, _ *string, locale *string) (*userdomain.User, error) {
	s.updates++
	user, _ := s.GetUserByID(42)
	if locale != nil {
		user.Locale = *locale
	}
	return user, nil
}

func profilePolicyRouter(service *profilePolicyService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user_id", uint(42)); c.Next() })
	h := NewUserProfileHandler(service)
	router.GET("/me", h.GetCurrentUser)
	router.PUT("/me", h.UpdateUserProfile)
	return router
}

func TestUnifiedProfileFieldPresenceRejectsNullAndMixedRequests(t *testing.T) {
	for _, field := range []string{"nickname", "email", "password", "avatar_url", "email_verified", "oauth_identities"} {
		t.Run(field, func(t *testing.T) {
			service := &profilePolicyService{policyErr: ErrUnifiedAuthRequired}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/me", strings.NewReader(`{"`+field+`":null,"locale":"en-US"}`))
			request.Header.Set("Content-Type", "application/json")
			profilePolicyRouter(service).ServeHTTP(recorder, request)
			if recorder.Code != http.StatusForbidden || service.updates != 0 {
				t.Fatalf("HTTP=%d writes=%d", recorder.Code, service.updates)
			}
		})
	}
}

func TestUnifiedProfileReadsAvatarAndAllowsLocale(t *testing.T) {
	service := &profilePolicyService{policyErr: ErrUnifiedAuthRequired}
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/me", strings.NewReader(`{"locale":"en-US"}`))
		request.Header.Set("Content-Type", "application/json")
		profilePolicyRouter(service).ServeHTTP(recorder, request)
		var body struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Data["avatar_url"] != "https://auth.example/avatar.png" {
			t.Fatalf("profile=%v", body.Data)
		}
		if method == http.MethodPut && (service.updates != 1 || body.Data["locale"] != "en-US") {
			t.Fatalf("profile=%v writes=%d", body.Data, service.updates)
		}
	}
}
