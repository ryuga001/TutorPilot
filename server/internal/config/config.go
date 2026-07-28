package config

import (
	"fmt"
	"os"
	"strconv"
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
	InviteTTL    time.Duration
	AppVerifyURL string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOPublicURL string
	MinIOUseSSL    bool

	LiveKitURL    string
	LiveKitKey    string
	LiveKitSecret string

	LiveKitRoomEmptyTimeout time.Duration
	LiveKitMaxParticipants  int

	LectureJoinTokenTTL time.Duration
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

		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "tutorpilot"),
		MinIOPublicURL: getEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
		MinIOUseSSL:    getBool("MINIO_USE_SSL", false),

		LiveKitURL:             getEnv("LIVEKIT_URL", "http://localhost:7880"),
		LiveKitKey:             getEnv("LIVEKIT_API_KEY", "tutorpilot"),
		LiveKitSecret:          getEnv("LIVEKIT_API_SECRET", "tutorpilot"),
		LiveKitMaxParticipants: getInt("LIVEKIT_MAX_PARTICIPANTS", 100),
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
	if cfg.InviteTTL, err = getDuration("INVITE_TTL", 72*time.Hour); err != nil {
		return nil, err
	}
	if cfg.LiveKitRoomEmptyTimeout, err = getDuration("LIVEKIT_ROOM_EMPTY_TIMEOUT", 5*time.Minute); err != nil {
		return nil, err
	}
	if cfg.LectureJoinTokenTTL, err = getDuration("LECTURE_JOIN_TOKEN_TTL", 2*time.Hour); err != nil {
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

func getBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func getInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
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
