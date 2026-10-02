package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Env            string
	HTTPPort       int
	RadiusAuthPort int
	RadiusAcctPort int
	DatabaseURL    string
	RedisAddr      string
	JWTSecret      string
	SecretsKey     string // 32-byte hex/base64 for AES-GCM credential encryption
	LogLevel       string
	DefaultLang    string
}

func Load() Config {
	return Config{
		Env:            env("ISP_ENV", "development"),
		HTTPPort:       envInt("HTTP_PORT", 8080),
		RadiusAuthPort: envInt("RADIUS_AUTH_PORT", 1812),
		RadiusAcctPort: envInt("RADIUS_ACCT_PORT", 1813),
		DatabaseURL:    env("DATABASE_URL", "postgres://isp:isp@localhost:5432/isp?sslmode=disable"),
		RedisAddr:      env("REDIS_ADDR", "localhost:6379"),
		JWTSecret:      env("JWT_SECRET", "dev-secret-change-me-please-32chars"),
		SecretsKey:     env("SECRETS_KEY", ""),
		LogLevel:       env("LOG_LEVEL", "info"),
		DefaultLang:    env("DEFAULT_LANG", "en"),
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}

func (c Config) Validate() error {
	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port %d", c.HTTPPort)
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET too short")
	}
	return nil
}
