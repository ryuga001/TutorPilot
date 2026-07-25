package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"workflow/internal/modules/notification"
	"workflow/internal/pkg/jwtutil"
	"workflow/internal/pkg/security"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired refresh token")
	ErrInvalidOTP         = errors.New("invalid or expired otp")
	ErrEmailNotVerified   = errors.New("email not verified; verify your email first")
)

type Service struct {
	repo       *Repository
	store      *RedisStore
	jwt        *jwtutil.Manager
	notifier   *notification.Notifier
	pepper     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	otpTTL     time.Duration
}

func NewService(
	repo *Repository,
	store *RedisStore,
	jwt *jwtutil.Manager,
	notifier *notification.Notifier,
	pepper string,
	accessTTL, refreshTTL, otpTTL time.Duration,
) *Service {
	return &Service{
		repo:       repo,
		store:      store,
		jwt:        jwt,
		notifier:   notifier,
		pepper:     pepper,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		otpTTL:     otpTTL,
	}
}

func (s *Service) SendVerification(ctx context.Context, email string) error {
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	otp, err := security.GenerateOTP(6)
	if err != nil {
		return err
	}
	if err := s.store.SaveEmailOTP(ctx, email, security.HashToken(otp), s.otpTTL); err != nil {
		return err
	}
	return s.notifier.SendEmailVerification(email, otp)
}

func (s *Service) VerifyEmail(ctx context.Context, email, otp string) error {
	stored, err := s.store.GetEmailOTP(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidOTP
	}
	if err != nil {
		return err
	}
	if stored != security.HashToken(otp) {
		return ErrInvalidOTP
	}
	if err := s.store.SetEmailVerified(ctx, email, s.otpTTL); err != nil {
		return err
	}
	return s.store.DeleteEmailOTP(ctx, email)
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	verified, err := s.store.IsEmailVerified(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, ErrEmailNotVerified
	}

	if _, err := s.repo.GetUserByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	salt, err := security.NewSalt()
	if err != nil {
		return nil, err
	}
	hash, err := security.HashPassword(req.Password, salt, s.pepper)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.CreateUser(ctx, req.Name, req.Email, hash, salt)
	if err != nil {
		return nil, err
	}

	_ = s.store.DeleteEmailVerified(ctx, req.Email)
	_ = s.store.DeleteEmailOTP(ctx, req.Email)

	return s.buildAuthResponse(ctx, user)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !security.CheckPassword(user.PasswordHash, req.Password, user.Salt, s.pepper) {
		return nil, ErrInvalidCredentials
	}
	return s.buildAuthResponse(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, raw string) (*TokenPair, error) {
	hash := security.HashToken(raw)
	userID, err := s.store.GetRefreshUser(ctx, hash)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	if err := s.store.DeleteRefresh(ctx, hash); err != nil {
		return nil, err
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user)
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	return s.store.DeleteRefresh(ctx, security.HashToken(raw))
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return err
	}
	if err != nil {
		return err
	}

	otp, err := security.GenerateOTP(6)
	if err != nil {
		return err
	}
	if err := s.store.SaveResetOTP(ctx, email, security.HashToken(otp), s.otpTTL); err != nil {
		return err
	}

	if err := s.notifier.SendPasswordReset(user.Name, user.Email, otp); err != nil {
		log.Printf("auth: failed to send reset email to %s: %v", user.Email, err)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	stored, err := s.store.GetResetOTP(ctx, req.Email)
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidOTP
	}
	if err != nil {
		return err
	}
	if stored != security.HashToken(req.OTP) {
		return ErrInvalidOTP
	}
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidOTP
		}
		return err
	}
	hash, err := security.HashPassword(req.NewPassword, user.Salt, s.pepper)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}
	return s.store.DeleteResetOTP(ctx, req.Email)
}

func (s *Service) GetMe(ctx context.Context, userID string) (*User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *Service) buildAuthResponse(ctx context.Context, user *User) (*AuthResponse, error) {
	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{User: user, Tokens: tokens}, nil
}

func (s *Service) issueTokens(ctx context.Context, user *User) (*TokenPair, error) {
	access, _, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}
	rawRefresh, refreshHash, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveRefresh(ctx, refreshHash, user.ID, s.refreshTTL); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}
