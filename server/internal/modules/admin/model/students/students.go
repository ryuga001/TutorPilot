package students

import (
	"errors"
	"time"

	"tutorpilot/internal/modules/admin/address"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("a student with this email already exists")
	ErrNoStudentRole      = errors.New("this organization has no Student role configured")
	ErrStorageUnavailable = errors.New("file storage is not configured")
)

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

type CreatedStudent struct {
	StudentView
	TempPassword string `json:"temp_password"`
}
