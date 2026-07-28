package students

import (
	"time"

	"tutorpilot/internal/pkg/address"
)

// Student is a dashboard_users row (identity + credentials) joined with the
// student-specific extras dashboard_users doesn't have. ID is the
// dashboard_users.id — a student's id, their JWT uid, and
// batch_students.student_id are all the same integer, since
// students.dashboard_user_id is the table's own primary key.
type Student struct {
	ID              int
	CustomerID      int
	FirstName       string
	LastName        string
	Email           string
	PhoneNo         string
	ProfileImageKey *string
	AddressID       *int
	CreatedBy       *int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type StudentView struct {
	ID              int           `json:"id"`
	FirstName       string        `json:"first_name"`
	LastName        string        `json:"last_name"`
	Email           string        `json:"email"`
	PhoneNo         string        `json:"phone_no"`
	ProfileImageURL string        `json:"profile_image_url,omitempty"`
	Address         *address.View `json:"address,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// CreatedStudent is the response to creating a student: the record plus the
// temporary password, shown exactly once and never on a read path.
type CreatedStudent struct {
	StudentView
	TempPassword string `json:"temp_password"`
}
