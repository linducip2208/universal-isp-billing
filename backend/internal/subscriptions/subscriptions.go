// Package subscriptions implements the subscriber lifecycle:
// pending -> active -> suspended -> active ... -> terminated.
package subscriptions

import (
	"errors"
	"time"
)

type Status string

const (
	Pending    Status = "pending"
	Active     Status = "active"
	Suspended  Status = "suspended"
	Terminated Status = "terminated"
)

type Subscription struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	CustomerID string    `json:"customer_id"`
	PackageID  string    `json:"package_id"`
	Username   string    `json:"username"`
	Service    string    `json:"service"` // pppoe | ipoe | hotspot | ftth
	Status     Status    `json:"status"`
	DeviceID   string    `json:"device_id,omitempty"`
	VLAN       int       `json:"vlan,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Subscription) Activate() error {
	if s.Status != Pending && s.Status != Suspended {
		return errors.New("only pending/suspended subscriptions can activate")
	}
	s.Status = Active
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Subscription) Suspend() error {
	if s.Status != Active {
		return errors.New("only active subscriptions can suspend")
	}
	s.Status = Suspended
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Subscription) Terminate() error {
	if s.Status == Terminated {
		return errors.New("already terminated")
	}
	s.Status = Terminated
	s.UpdatedAt = time.Now().UTC()
	return nil
}
