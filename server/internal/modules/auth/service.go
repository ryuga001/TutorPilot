package auth

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/jwtutil"
	"tutorpilot/internal/pkg/security"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrOrgTaken           = errors.New("organization name already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired refresh token")
	ErrInvalidOTP         = errors.New("invalid or expired otp")
	ErrEmailNotVerified   = errors.New("email not verified; verify your email first")
)

const (
	secretCacheTTL = 30 * time.Minute
	privCacheTTL   = 5 * time.Minute
)

type Service struct {
	repo       *Repository
	store      *RedisStore
	jwt        *jwtutil.Manager
	notifier   *notification.Notifier
	pepper     string
	refreshTTL time.Duration
	otpTTL     time.Duration
}

func NewService(
	repo *Repository,
	store *RedisStore,
	jwt *jwtutil.Manager,
	notifier *notification.Notifier,
	pepper string,
	refreshTTL, otpTTL time.Duration,
) *Service {
	return &Service{
		repo:       repo,
		store:      store,
		jwt:        jwt,
		notifier:   notifier,
		pepper:     pepper,
		refreshTTL: refreshTTL,
		otpTTL:     otpTTL,
	}
}

func (s *Service) SendVerification(ctx context.Context, email string) error {
	email = NormalizeEmail(email)

	taken, err := s.repo.EmailTaken(ctx, email)
	if err != nil {
		return err
	}
	if taken {
		return ErrEmailTaken
	}

	otp, err := security.GenerateOTP(6)
	if err != nil {
		return err
	}
	if err := s.store.SaveEmailOTP(ctx, email, security.HashToken(otp), s.otpTTL); err != nil {
		return err
	}
	return s.notifier.SendEmailVerification(ctx, email, otp)
}

func (s *Service) VerifyEmail(ctx context.Context, email, otp string) error {
	email = NormalizeEmail(email)

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

// --- Registration (tenant onboarding) ----------------------------------------

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	email := NormalizeEmail(req.Email)

	verified, err := s.store.IsEmailVerified(ctx, email)
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, ErrEmailNotVerified
	}

	salt, err := security.NewSalt()
	if err != nil {
		return nil, err
	}
	hash, err := security.HashPassword(req.Password, salt, s.pepper)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.CreateTenantWithAdmin(ctx, req.OrgName, req.FirstName, req.LastName, email, hash, salt)
	if err != nil {
		return nil, err
	}

	_ = s.store.DeleteEmailVerified(ctx, email)
	_ = s.store.DeleteEmailOTP(ctx, email)

	return s.buildAuthResponse(ctx, user)
}

// --- Login / tokens ----------------------------------------------------------

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !security.CheckPassword(user.PasswordHash, req.Password, user.PasswordSalt, s.pepper) {
		return nil, ErrInvalidCredentials
	}
	return s.buildAuthResponse(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, raw string) (*TokenPair, error) {
	hash := security.HashToken(raw)
	stored, err := s.store.GetRefreshUser(ctx, hash)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	userID, err := strconv.Atoi(stored)
	if err != nil {
		return nil, ErrInvalidToken
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

// --- Password reset ----------------------------------------------------------

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		// Never disclose whether the address exists.
		return nil
	}
	if err != nil {
		return err
	}

	otp, err := security.GenerateOTP(6)
	if err != nil {
		return err
	}
	if err := s.store.SaveResetOTP(ctx, user.Email, security.HashToken(otp), s.otpTTL); err != nil {
		return err
	}
	name := user.FirstName
	if name == "" {
		name = user.Email
	}
	if err := s.notifier.SendPasswordReset(ctx, name, user.Email, otp); err != nil {
		log.Printf("auth: failed to send reset email to %s: %v", user.Email, err)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	email := NormalizeEmail(req.Email)

	stored, err := s.store.GetResetOTP(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidOTP
	}
	if err != nil {
		return err
	}
	if stored != security.HashToken(req.OTP) {
		return ErrInvalidOTP
	}
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidOTP
		}
		return err
	}
	if err := s.setPassword(ctx, user, req.NewPassword); err != nil {
		return err
	}
	return s.store.DeleteResetOTP(ctx, email)
}

// ChangePassword lets a signed-in user change their own password.
func (s *Service) ChangePassword(ctx context.Context, userID int, req ChangePasswordRequest) (*AuthResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !security.CheckPassword(user.PasswordHash, req.CurrentPassword, user.PasswordSalt, s.pepper) {
		return nil, ErrInvalidCredentials
	}
	if err := s.setPassword(ctx, user, req.NewPassword); err != nil {
		return nil, err
	}
	fresh, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildAuthResponse(ctx, fresh)
}

func (s *Service) setPassword(ctx context.Context, user *DashboardUser, password string) error {
	hash, err := security.HashPassword(password, user.PasswordSalt, s.pepper)
	if err != nil {
		return err
	}
	return s.repo.SetPassword(ctx, user.ID, hash)
}

func (s *Service) GetMe(ctx context.Context, userID int) (*UserView, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	view := user.View()
	if privs, err := s.GetPrivileges(ctx, userID); err == nil {
		view.Privileges = privs
	}
	return view, nil
}

// GetPrivileges returns the user's privilege names, caching the set in redis.
func (s *Service) GetPrivileges(ctx context.Context, userID int) ([]string, error) {
	privs, err := s.store.GetPrivileges(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		privs, err = s.repo.GetUserPrivileges(ctx, userID)
		if err != nil {
			return nil, err
		}
		_ = s.store.CachePrivileges(ctx, userID, privs, privCacheTTL)
	} else if err != nil {
		return nil, err
	}
	return privs, nil
}

// --- Middleware resolvers -----------------------------------------------------

// ResolveSecret returns a tenant's JWT signing secret, caching it in redis.
func (s *Service) ResolveSecret(ctx context.Context, customerID int) ([]byte, error) {
	if cached, err := s.store.GetJWTSecret(ctx, customerID); err == nil {
		return []byte(cached), nil
	}
	secret, err := s.repo.GetCustomerJWTSecret(ctx, customerID)
	if err != nil {
		return nil, err
	}
	_ = s.store.CacheJWTSecret(ctx, customerID, secret, secretCacheTTL)
	return []byte(secret), nil
}

func (s *Service) HasPrivilege(ctx context.Context, userIDStr, privilege string) (bool, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return false, err
	}
	privs, err := s.GetPrivileges(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range privs {
		if p == privilege {
			return true, nil
		}
	}
	return false, nil
}

// --- internals ---------------------------------------------------------------

func (s *Service) buildAuthResponse(ctx context.Context, user *DashboardUser) (*AuthResponse, error) {
	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{User: user.View(), Tokens: tokens}, nil
}

func (s *Service) issueTokens(ctx context.Context, user *DashboardUser) (*TokenPair, error) {
	secret, err := s.ResolveSecret(ctx, user.CustomerID)
	if err != nil {
		return nil, err
	}
	access, _, err := s.jwt.Generate(secret, jwtutil.Identity{
		UserID:     strconv.Itoa(user.ID),
		Email:      user.Email,
		CustomerID: user.CustomerID,
		Role:       user.RoleType,
	})
	if err != nil {
		return nil, err
	}
	rawRefresh, refreshHash, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveRefresh(ctx, refreshHash, strconv.Itoa(user.ID), s.refreshTTL); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwt.AccessTTL().Seconds()),
	}, nil
}
