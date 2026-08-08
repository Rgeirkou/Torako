package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("ADDR", ":9999")
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Addr != ":9999" {
		t.Errorf("Addr = %q, want :9999", cfg.Addr)
	}
	if cfg.Env != "test" {
		t.Errorf("Env = %q, want test", cfg.Env)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want 15s", cfg.ReadTimeout)
	}
}

func TestLoad_Validation(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		want string
	}{
		{
			name: "member max zero",
			envs: map[string]string{"RATE_LIMIT_MEMBER_MAX": "0"},
			want: "RATE_LIMIT_MEMBER_MAX",
		},
		{
			name: "partner max negative",
			envs: map[string]string{"RATE_LIMIT_PARTNER_MAX": "-1"},
			want: "RATE_LIMIT_PARTNER_MAX",
		},
		{
			name: "ip max zero",
			envs: map[string]string{"RATE_LIMIT_IP_MAX": "0"},
			want: "RATE_LIMIT_IP_MAX",
		},
		{
			name: "member max zero when disabled",
			envs: map[string]string{"RATE_LIMIT_ENABLED": "false", "RATE_LIMIT_MEMBER_MAX": "0"},
			want: "",
		},
		{
			name: "zero window",
			envs: map[string]string{"RATE_LIMIT_WINDOW": "0s"},
			want: "RATE_LIMIT_WINDOW",
		},
		{
			name: "production without bootstrap key",
			envs: map[string]string{"APP_ENV": "production", "BOOTSTRAP_API_KEY": ""},
			want: "BOOTSTRAP_API_KEY",
		},
		{
			name: "production with bootstrap key",
			envs: map[string]string{"APP_ENV": "production", "BOOTSTRAP_API_KEY": "seed-key"},
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			cfg, err := Load()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("load config: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error mentioning %s, got config %+v", tc.want, cfg)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("default Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.RateLimitMemberMax != 60 {
		t.Errorf("RateLimitMemberMax = %d, want 60", cfg.RateLimitMemberMax)
	}
	if cfg.RateLimitPartnerMax != 1000 {
		t.Errorf("RateLimitPartnerMax = %d, want 1000", cfg.RateLimitPartnerMax)
	}
	if cfg.RateLimitIPMax != 1000 {
		t.Errorf("RateLimitIPMax = %d, want 1000", cfg.RateLimitIPMax)
	}
	if cfg.TrustProxyHeaders {
		t.Error("TrustProxyHeaders must default to false")
	}
}
