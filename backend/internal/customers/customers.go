// Package customers implements customer, group, address, and contact models
// with validation and status transitions.
package customers

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusClosed    Status = "closed"
)

type Customer struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Status    Status    `json:"status"`
	GroupID   string    `json:"group_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Group struct {
	ID    string `json:"id"`
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
}

type Address struct {
	ID         string  `json:"id"`
	CustomerID string  `json:"customer_id"`
	Label      string  `json:"label"`
	Address    string  `json:"address"`
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
}

type Contact struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Kind       string `json:"kind"` // email | phone | whatsapp
	Value      string `json:"value"`
}

func Validate(c *Customer) error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name required")
	}
	if c.Email != "" {
		if _, err := mail.ParseAddress(c.Email); err != nil {
			return errors.New("invalid email")
		}
	}
	if c.Phone != "" && len(c.Phone) < 7 {
		return errors.New("invalid phone")
	}
	return nil
}

// Transition enforces active <-> suspended -> closed lifecycle.
func Transition(c *Customer, to Status) error {
	switch {
	case c.Status == StatusActive && (to == StatusSuspended || to == StatusClosed):
		c.Status = to
	case c.Status == StatusSuspended && (to == StatusActive || to == StatusClosed):
		c.Status = to
	case c.Status == to:
		return nil
	default:
		return errors.New("illegal customer status transition")
	}
	c.UpdatedAt = time.Now().UTC()
	return nil
}
