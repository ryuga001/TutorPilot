package auth

import "time"

// DashboardUser is a tenant principal — an admin, a tutor or a student; all three
// are plain rows in this table (tutors/students additionally have an extras row
// in their own table, keyed by this id — see migration 000012). Sensitive fields
// are never serialised.
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

// UserView is the public shape returned by the auth endpoints.
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
