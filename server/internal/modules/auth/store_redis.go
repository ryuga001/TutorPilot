package auth

import (
	"context"
	"errors"
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
