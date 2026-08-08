package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Addr                string        `env:"ADDR" envDefault:":8080"`
	Env                 string        `env:"APP_ENV" envDefault:"development"`
	DatabaseURL         string        `env:"DATABASE_URL"`
	AllowOrigins        []string      `env:"ALLOW_ORIGINS" envSeparator:","`
	ReadTimeout         time.Duration `env:"READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout        time.Duration `env:"WRITE_TIMEOUT" envDefault:"35s"`
	IdleTimeout         time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout     time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	TrueMoneyBaseURL    string        `env:"TRUEMONEY_API_BASE_URL" envDefault:"https://gift.truemoney.com"`
	TrueMoneyTimeout    time.Duration `env:"TRUEMONEY_API_TIMEOUT" envDefault:"30s"`
	BootstrapAPIKey     string        `env:"BOOTSTRAP_API_KEY"`
	RateLimitEnabled    bool          `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RateLimitMemberMax  int           `env:"RATE_LIMIT_MEMBER_MAX" envDefault:"60"`
	RateLimitPartnerMax int           `env:"RATE_LIMIT_PARTNER_MAX" envDefault:"1000"`
	RateLimitWindow     time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	RateLimitIPEnabled  bool          `env:"RATE_LIMIT_IP_ENABLED" envDefault:"true"`
	RateLimitIPMax      int           `env:"RATE_LIMIT_IP_MAX" envDefault:"1000"`
	RateLimitIPWindow   time.Duration `env:"RATE_LIMIT_IP_WINDOW" envDefault:"1m"`
	TrustProxyHeaders   bool          `env:"TRUST_PROXY_HEADERS" envDefault:"false"`
	RedisURL            string        `env:"REDIS_URL"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate rejects configurations that would break rate limiting (e.g. a
// max of 0 would deny every request) before the server starts.
func (c *Config) validate() error {
	require := func(name string, max int, enabled bool) error {
		if enabled && max < 1 {
			return fmt.Errorf("config: %s must be at least 1 when enabled", name)
		}
		return nil
	}
	if err := require("RATE_LIMIT_MEMBER_MAX", c.RateLimitMemberMax, c.RateLimitEnabled); err != nil {
		return err
	}
	if err := require("RATE_LIMIT_PARTNER_MAX", c.RateLimitPartnerMax, c.RateLimitEnabled); err != nil {
		return err
	}
	if err := require("RATE_LIMIT_IP_MAX", c.RateLimitIPMax, c.RateLimitIPEnabled); err != nil {
		return err
	}
	if c.RateLimitEnabled && c.RateLimitWindow <= 0 {
		return fmt.Errorf("config: RATE_LIMIT_WINDOW must be positive when rate limiting is enabled")
	}
	if c.RateLimitIPEnabled && c.RateLimitIPWindow <= 0 {
		return fmt.Errorf("config: RATE_LIMIT_IP_WINDOW must be positive when IP rate limiting is enabled")
	}
	if c.Env == "production" && c.BootstrapAPIKey == "" {
		return fmt.Errorf("config: BOOTSTRAP_API_KEY is required when APP_ENV=production (a silently generated key would be impossible to recover)")
	}
	return nil
}
