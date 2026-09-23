package service

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/agridispatch/agridispatch/internal/model"
)

func newTestAuthService() *AuthService {
	return NewAuthService(nil, "unit-test-secret-which-is-long-enough", 1, slog.Default())
}

func TestGenerateAndParseToken(t *testing.T) {
	svc := newTestAuthService()
	tests := []struct {
		name string
		user model.User
	}{
		{name: "admin", user: model.User{ID: 1, Username: "admin", Role: "admin"}},
		{name: "user", user: model.User{ID: 2, Username: "zhangwei", Role: "user"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateToken(&tt.user)
			if err != nil {
				t.Fatalf("GenerateToken: %v", err)
			}
			claims, err := svc.ParseToken(token)
			if err != nil {
				t.Fatalf("ParseToken: %v", err)
			}
			if claims.UserID != tt.user.ID || claims.Role != tt.user.Role || claims.Subject != tt.user.Username {
				t.Fatalf("claims mismatch: %+v", claims)
			}
		})
	}
}

func TestParseTokenRejectsInvalid(t *testing.T) {
	svc := newTestAuthService()
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty", token: ""},
		{name: "garbage", token: "not-a-jwt"},
		{name: "wrong signature", token: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhZG1pbiJ9.invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.ParseToken(tt.token); err == nil {
				t.Fatalf("expected error for %q", tt.token)
			}
		})
	}
}

func TestParseTokenRejectsExpired(t *testing.T) {
	svc := NewAuthService(nil, "unit-test-secret-which-is-long-enough", 0, slog.Default())
	token, err := svc.jwtSignWithClaims(time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := svc.ParseToken(token); err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("expected expired token error, got %v", err)
	}
}
