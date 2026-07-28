package tutors

import (
	"time"

	"tutorpilot/internal/pkg/address"
)

// Tutor is a dashboard_users row (identity + credentials) joined with the
// tutor-specific extras dashboard_users doesn't have. ID is the
// dashboard_users.id — a tutor's id, their JWT uid, and batch_tutors.tutor_id /
// lectures.tutor_id are all the same integer, since tutors.dashboard_user_id is
// the table's own primary key.
type Tutor struct {
	ID              int
	CustomerID      int
	FirstName       string
	LastName        string
	Email           string
	PhoneNo         string
	Designation     string
	ProfileImageKey *string
	AddressID       *int
	CreatedBy       *int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TutorView struct {
	ID              int           `json:"id"`
	FirstName       string        `json:"first_name"`
	LastName        string        `json:"last_name"`
	Email           string        `json:"email"`
	PhoneNo         string        `json:"phone_no"`
	Designation     string        `json:"designation"`
	ProfileImageURL string        `json:"profile_image_url,omitempty"`
	Address         *address.View `json:"address,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// CreatedTutor is the response to creating a tutor: the record plus the
// temporary password, shown exactly once and never on a read path.
type CreatedTutor struct {
	TutorView
	TempPassword string `json:"temp_password"`
}
