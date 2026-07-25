package auth

import "time"

// DashboardUser is a tenant user row joined with its role type and the tenant's
// contact name (used for personalised emails). Sensitive fields are never
// serialised.
type DashboardUser struct {
	ID           int    `json:"id"`
	CustomerID   int    `json:"customer_id"`
	RoleID       *int   `json:"role_id"`
	RoleType     string `json:"role"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	PasswordSalt string `json:"-"`
	FirstName    string `json:"-"`

	CreatedAt time.Time `json:"created_at"`
}

// UserView is the public shape returned by the auth endpoints.
type UserView struct {
	ID         int       `json:"id"`
	CustomerID int       `json:"customer_id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	Privileges []string  `json:"privileges,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (u *DashboardUser) View() *UserView {
	return &UserView{
		ID:         u.ID,
		CustomerID: u.CustomerID,
		Email:      u.Email,
		Role:       u.RoleType,
		CreatedAt:  u.CreatedAt,
	}
}
