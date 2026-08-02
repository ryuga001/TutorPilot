package tutors

import (
	"errors"
	"time"

	"tutorpilot/internal/modules/admin/address"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("a tutor with this email already exists")
	ErrNoTutorRole        = errors.New("this organization has no Tutor role configured")
	ErrStorageUnavailable = errors.New("file storage is not configured")
)

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

type CreatedTutor struct {
	TutorView
	TempPassword string `json:"temp_password"`
}
