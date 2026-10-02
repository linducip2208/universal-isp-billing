// Package subscriptions implements the auditable subscriber lifecycle:
// prospect -> pending -> provisioning -> active <-> suspended <-> grace ->
// reactivating -> active ... -> terminated|cancelled, with provision_failed
// as an explicit failure state. Every transition is explicit.
package subscriptions

import (
	"errors"
	"time"
)

type Status string

const (
	Prospect        Status = "prospect"
	Pending         Status = "pending"
	Provisioning    Status = "provisioning"
	ProvisionFailed Status = "provision_failed"
	Active          Status = "active"
	Suspended       Status = "suspended"
	Grace           Status = "grace"
	Reactivating    Status = "reactivating"
	Terminated      Status = "terminated"
	Cancelled       Status = "cancelled"
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

var transitions = map[Status][]Status{
	Prospect:        {Pending, Cancelled},
	Pending:         {Provisioning, Cancelled},
	Provisioning:    {Active, ProvisionFailed, Cancelled},
	ProvisionFailed: {Provisioning, Cancelled},
	Active:          {Suspended, Grace, Terminated, Cancelled},
	Suspended:       {Reactivating, Grace, Terminated, Cancelled},
	Grace:           {Reactivating, Suspended, Terminated},
	Reactivating:    {Provisioning, Active, Suspended},
	Terminated:      {},
	Cancelled:       {},
}

// Transition moves the subscription; illegal moves error (auditable by callers).
func (s *Subscription) Transition(to Status) error {
	if s.Status == to {
		return nil
	}
	for _, ok := range transitions[s.Status] {
		if ok == to {
			s.Status = to
			s.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("illegal subscription transition")
}

func (s *Subscription) Activate() error {
	if s.Status == Pending || s.Status == Provisioning {
		return s.Transition(Active)
	}
	return s.Transition(Reactivating)
}

func (s *Subscription) Suspend() error { return s.Transition(Suspended) }

func (s *Subscription) Terminate() error {
	if s.Status == Terminated {
		return errors.New("already terminated")
	}
	if s.Status == Cancelled {
		return errors.New("already cancelled")
	}
	// Terminated reachable from most states; cancelled only pre-service.
	if s.Status == Prospect || s.Status == Pending || s.Status == Provisioning || s.Status == ProvisionFailed {
		return errors.New("use Cancelled before service, Terminate after")
	}
	return s.Transition(Terminated)
}
