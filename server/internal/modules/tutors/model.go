package tutors

import (
	"time"

	"tutorpilot/internal/pkg/address"
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
