package students

import "tutorpilot/internal/modules/admin/address"

type CreateStudentRequest struct {
	FirstName string         `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string         `json:"last_name" binding:"required,min=1,max=100"`
	Email     string         `json:"email" binding:"required,email,max=150"`
	PhoneNo   string         `json:"phone_no" binding:"max=30"`
	Address   *address.Input `json:"address"`
}

type UpdateStudentRequest struct {
	FirstName string         `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string         `json:"last_name" binding:"required,min=1,max=100"`
	Email     string         `json:"email" binding:"required,email,max=150"`
	PhoneNo   string         `json:"phone_no" binding:"max=30"`
	Address   *address.Input `json:"address"`
}
