package models

import (
	"fmt"
	"regexp"
	"time"
)

// Company represents a company in the system
type Company struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	NIT            int64     `json:"nit"`
	NCR            int64     `json:"ncr"`
	HCUsername     string    `json:"hc_username"`
	HCPasswordRef  string    `json:"-"` // Never expose in JSON
	LastActivityAt time.Time `json:"last_activity_at"`
	Email          string    `json:"email"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateCompanyRequest represents the request to create a company
type CreateCompanyRequest struct {
	Name       string `json:"name" binding:"required"`
	NIT        int64  `json:"nit" binding:"required"`
	NCR        int64  `json:"ncr" binding:"required"`
	HCUsername string `json:"hc_username" binding:"required"`
	HCPassword string `json:"hc_password" binding:"required"`
	Email      string `json:"email" binding:"required"`
}

// Validate validates the create company request
func (r *CreateCompanyRequest) Validate() error {
	// Validate name
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate NIT (must be positive)
	if r.NIT <= 0 {
		return fmt.Errorf("nit must be a positive number")
	}

	// Validate NCR (must be positive)
	if r.NCR <= 0 {
		return fmt.Errorf("ncr must be a positive number")
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(r.Email) {
		return fmt.Errorf("email format is invalid")
	}

	// Validate username
	if r.HCUsername == "" {
		return fmt.Errorf("hc_username is required")
	}

	// Validate password
	if r.HCPassword == "" {
		return fmt.Errorf("hc_password is required")
	}

	return nil
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
