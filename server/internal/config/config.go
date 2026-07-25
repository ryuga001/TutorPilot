package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DatabaseURL string
	RedisURL    string

	CORSAllowedOrigins []string

	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	PasswordPepper string

	SMTPHost string
	SMTPPort string
	SMTPFrom string

	OTPTTL       time.Duration
	AppVerifyURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		AppPort:            getEnv("APP_PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		PasswordPepper:     getEnv("PASSWORD_PEPPER", ""),
		SMTPHost:           getEnv("SMTP_HOST", "localhost"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		SMTPFrom:           getEnv("SMTP_FROM", "TutorPilot <no-reply@tutorpilot.ai>"),
		AppVerifyURL:       getEnv("APP_VERIFY_URL", "http://localhost:8080/verify-email"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "*")),
	}

	var err error
	if cfg.JWTAccessTTL, err = getDuration("JWT_ACCESS_TTL", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.JWTRefreshTTL, err = getDuration("JWT_REFRESH_TTL", 168*time.Hour); err != nil {
		return nil, err
	}
	if cfg.OTPTTL, err = getDuration("OTP_TTL", 5*time.Minute); err != nil {
		return nil, err
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.PasswordPepper == "" {
		return nil, fmt.Errorf("PASSWORD_PEPPER is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}
