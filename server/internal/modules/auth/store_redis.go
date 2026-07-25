package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func refreshKey(tokenHash string) string { return "refresh:" + tokenHash }
func emailOTPKey(email string) string    { return "email_otp:" + email }
func verifiedKey(email string) string    { return "email_verified:" + email }
func resetKey(email string) string       { return "pwd_reset:" + email }
func secretKey(customerID int) string    { return "jwt_secret:" + strconv.Itoa(customerID) }
func privKey(userID int) string          { return "privileges:" + strconv.Itoa(userID) }

func (s *RedisStore) SaveRefresh(ctx context.Context, tokenHash, userID string, ttl time.Duration) error {
	return s.rdb.Set(ctx, refreshKey(tokenHash), userID, ttl).Err()
}

func (s *RedisStore) GetRefreshUser(ctx context.Context, tokenHash string) (string, error) {
	userID, err := s.rdb.Get(ctx, refreshKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *RedisStore) DeleteRefresh(ctx context.Context, tokenHash string) error {
	return s.rdb.Del(ctx, refreshKey(tokenHash)).Err()
}

func (s *RedisStore) SaveEmailOTP(ctx context.Context, email, otpHash string, ttl time.Duration) error {
	return s.rdb.Set(ctx, emailOTPKey(email), otpHash, ttl).Err()
}

func (s *RedisStore) GetEmailOTP(ctx context.Context, email string) (string, error) {
	otpHash, err := s.rdb.Get(ctx, emailOTPKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return otpHash, nil
}

func (s *RedisStore) DeleteEmailOTP(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, emailOTPKey(email)).Err()
}

func (s *RedisStore) SetEmailVerified(ctx context.Context, email string, ttl time.Duration) error {
	return s.rdb.Set(ctx, verifiedKey(email), "1", ttl).Err()
}

func (s *RedisStore) IsEmailVerified(ctx context.Context, email string) (bool, error) {
	_, err := s.rdb.Get(ctx, verifiedKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *RedisStore) DeleteEmailVerified(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, verifiedKey(email)).Err()
}

func (s *RedisStore) SaveResetOTP(ctx context.Context, email, otpHash string, ttl time.Duration) error {
	return s.rdb.Set(ctx, resetKey(email), otpHash, ttl).Err()
}

func (s *RedisStore) GetResetOTP(ctx context.Context, email string) (string, error) {
	otpHash, err := s.rdb.Get(ctx, resetKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return otpHash, nil
}

func (s *RedisStore) DeleteResetOTP(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, resetKey(email)).Err()
}

// --- Per-request auth caches (best-effort; DB is the source of truth) --------

func (s *RedisStore) CacheJWTSecret(ctx context.Context, customerID int, secret string, ttl time.Duration) error {
	return s.rdb.Set(ctx, secretKey(customerID), secret, ttl).Err()
}

func (s *RedisStore) GetJWTSecret(ctx context.Context, customerID int) (string, error) {
	secret, err := s.rdb.Get(ctx, secretKey(customerID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return secret, nil
}

func (s *RedisStore) CachePrivileges(ctx context.Context, userID int, privs []string, ttl time.Duration) error {
	b, err := json.Marshal(privs)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, privKey(userID), b, ttl).Err()
}

func (s *RedisStore) GetPrivileges(ctx context.Context, userID int) ([]string, error) {
	raw, err := s.rdb.Get(ctx, privKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var privs []string
	if err := json.Unmarshal(raw, &privs); err != nil {
		return nil, err
	}
	return privs, nil
}
