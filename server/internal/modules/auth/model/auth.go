package auth

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrOrgTaken           = errors.New("organization name already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired refresh token")
	ErrInvalidOTP         = errors.New("invalid or expired otp")
	ErrEmailNotVerified   = errors.New("email not verified; verify your email first")
)

type DashboardUser struct {
	ID           int    `json:"id"`
	CustomerID   int    `json:"customer_id"`
	RoleID       *int   `json:"role_id"`
	RoleType     string `json:"role"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	PasswordSalt string `json:"-"`
	FirstName    string `json:"-"`
	LastName     string `json:"-"`

	CreatedAt time.Time `json:"created_at"`
}

type UserView struct {
	ID         int       `json:"id"`
	CustomerID int       `json:"customer_id"`
	Email      string    `json:"email"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Role       string    `json:"role"`
	Privileges []string  `json:"privileges,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (u *DashboardUser) View() *UserView {
	return &UserView{
		ID:         u.ID,
		CustomerID: u.CustomerID,
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Role:       u.RoleType,
		CreatedAt:  u.CreatedAt,
	}
}
