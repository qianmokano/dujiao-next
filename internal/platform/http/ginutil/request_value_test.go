package ginutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHasJSONFieldMatchesJSONDecoder(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"canonical null", `{"username":null}`, true},
		{"ASCII case", `{"Username":"value"}`, true},
		{"Unicode long s", `{"uſername":null}`, true},
		{"Unicode password", `{"paſſword":null}`, true},
		{"Unicode Kelvin", `{"nicKname":null}`, true},
		{"false value", `{"AVATAR_URL":false}`, true},
		{"case duplicate null", `{"username":"value","USERNAME":null}`, true},
		{"business only", `{"balance_notify_enabled":false}`, false},
		{"no trimming", `{" username ":null}`, false},
		{"different Unicode letter", `{"nıckname":null}`, false},
		{"empty", `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var fields map[string]json.RawMessage
			var decoded struct {
				Username  json.RawMessage `json:"username"`
				Nickname  json.RawMessage `json:"nickname"`
				Password  json.RawMessage `json:"password"`
				AvatarURL json.RawMessage `json:"avatar_url"`
			}
			if err := json.Unmarshal([]byte(tc.body), &fields); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.body), &decoded); err != nil {
				t.Fatal(err)
			}
			bound := decoded.Username != nil || decoded.Nickname != nil || decoded.Password != nil || decoded.AvatarURL != nil
			got := HasJSONField(fields, "username", "nickname", "password", "avatar_url")
			if got != tc.want || got != bound {
				t.Fatalf("present=%v decoder=%v want=%v", got, bound, tc.want)
			}
		})
	}
	if HasJSONField(nil, "username") || HasJSONField(map[string]json.RawMessage{"username": nil}) {
		t.Fatal("missing fields or names must not match")
	}
}

func TestParseQueryBoolPtr(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		target     string
		wantNil    bool
		want       bool
		wantErr    bool
		queryParam string
	}{
		{
			name:       "missing query returns nil",
			target:     "/",
			wantNil:    true,
			queryParam: "is_active",
		},
		{
			name:       "blank query returns nil",
			target:     "/?is_active=+",
			wantNil:    true,
			queryParam: "is_active",
		},
		{
			name:       "parses trimmed true",
			target:     "/?is_active=+true+",
			want:       true,
			queryParam: "is_active",
		},
		{
			name:       "parses false",
			target:     "/?is_active=false",
			want:       false,
			queryParam: "is_active",
		},
		{
			name:       "rejects invalid bool",
			target:     "/?is_active=maybe",
			wantErr:    true,
			queryParam: "is_active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, tt.target, nil)

			got, err := ParseQueryBoolPtr(c, tt.queryParam)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseQueryBoolPtr: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil bool, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected bool value, got nil")
			}
			if *got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, *got)
			}
		})
	}
}

func TestParseOptionalBoolValue(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantNil bool
		want    bool
		wantErr bool
	}{
		{name: "empty value returns nil", raw: "", wantNil: true},
		{name: "blank value returns nil", raw: "  ", wantNil: true},
		{name: "parses trimmed true", raw: " true ", want: true},
		{name: "parses false", raw: "false", want: false},
		{name: "rejects invalid bool", raw: "maybe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOptionalBoolValue(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseOptionalBoolValue: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil bool, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected bool value, got nil")
			}
			if *got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, *got)
			}
		})
	}
}

func TestParseQueryBoolDefaultsMissingAndBlankToFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		target  string
		want    bool
		wantErr bool
	}{
		{name: "missing query returns false", target: "/", want: false},
		{name: "blank query returns false", target: "/?force_refresh=+", want: false},
		{name: "parses true", target: "/?force_refresh=true", want: true},
		{name: "rejects invalid bool", target: "/?force_refresh=nope", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, tt.target, nil)

			got, err := ParseQueryBool(c, "force_refresh")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseQueryBool: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
