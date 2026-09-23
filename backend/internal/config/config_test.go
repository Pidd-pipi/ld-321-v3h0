package config

import (
	"strings"
	"testing"
)

func TestDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{name: "default", cfg: Config{DBUser: "u", DBPassword: "p", DBHost: "db", DBPort: "3306", DBName: "agridispatch"}, want: "u:p@tcp(db:3306)/agridispatch?charset=utf8mb4&parseTime=True&loc=Local"},
		{name: "custom", cfg: Config{DBUser: "root", DBPassword: "secret", DBHost: "127.0.0.1", DBPort: "33321", DBName: "db2"}, want: "root:secret@tcp(127.0.0.1:33321)/db2?charset=utf8mb4&parseTime=True&loc=Local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.DSN(); got != tt.want {
				t.Fatalf("DSN()=%q want %q", got, tt.want)
			}
		})
	}
}

func TestRedisAddr(t *testing.T) {
	cfg := Config{RedisHost: "redis", RedisPort: "6379"}
	if got := cfg.RedisAddr(); got != "redis:6379" {
		t.Fatalf("RedisAddr()=%q want redis:6379", got)
	}
}

func TestCORSOrigins(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "single", in: "*", want: []string{"*"}},
		{name: "multi", in: "http://a.com, http://b.com", want: []string{"http://a.com", "http://b.com"}},
		{name: "spaces", in: "  http://a.com ,, http://b.com ", want: []string{"http://a.com", "http://b.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (&Config{CORSAllowedOrigins: tt.in}).CORSOrigins()
			if len(got) != len(tt.want) {
				t.Fatalf("CORSOrigins()=%v want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("CORSOrigins()=%v want %v", got, tt.want)
				}
			}
		})
	}
}

func TestLoadRejectsWeakSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "change_me_to_a_long_random_string")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://example.com")
	t.Setenv("AUTH_RATE_LIMIT", "10")
	t.Setenv("AUTH_RATE_WINDOW_SECONDS", "60")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected weak secret rejection, got %v", err)
	}
}

func TestLoadRejectsWildcardCORSInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 40))
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	t.Setenv("AUTH_RATE_LIMIT", "10")
	t.Setenv("AUTH_RATE_WINDOW_SECONDS", "60")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
		t.Fatalf("expected wildcard CORS rejection, got %v", err)
	}
}

func TestLoadAcceptsProductionConfig(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", strings.Repeat("y", 40))
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:18621")
	t.Setenv("AUTH_RATE_LIMIT", "10")
	t.Setenv("AUTH_RATE_WINDOW_SECONDS", "60")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
	if cfg.JWTSecret != strings.Repeat("y", 40) {
		t.Fatalf("JWTSecret mismatch")
	}
}
