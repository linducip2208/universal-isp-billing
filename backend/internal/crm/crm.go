// Package crm: lead pipeline with an auditable, validated lifecycle:
// lead -> prospect -> surveyed -> quoted -> installation -> won|lost.
// Conversion to customer/subscription is explicit (Convert records IDs).
package crm

import (
	"errors"
	"time"
)

type Stage string

const (
	StageLead    Stage = "lead"
	Prospect     Stage = "prospect"
	Surveyed     Stage = "surveyed"
	Quoted       Stage = "quoted"
	Installation Stage = "installation"
	Won          Stage = "won"
	Lost         Stage = "lost"
)

type Lead struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone,omitempty"`
	Address     string    `json:"address,omitempty"`
	Stage       Stage     `json:"stage"`
	QuotedCents int64     `json:"quoted_cents"`
	CustomerID  string    `json:"customer_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var pipeline = map[Stage][]Stage{
	StageLead:    {Prospect, Lost},
	Prospect:     {Surveyed, Lost},
	Surveyed:     {Quoted, Lost},
	Quoted:       {Installation, Lost},
	Installation: {Won, Lost},
	Won:          {},
	Lost:         {StageLead}, // re-engagement reopens at lead
}

func (l *Lead) Advance(to Stage) error {
	if l.Stage == to {
		return nil
	}
	for _, ok := range pipeline[l.Stage] {
		if ok == to {
			l.Stage = to
			l.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return errors.New("illegal pipeline transition")
}

// Convert records the won customer/subscription (called after provisioning).
func (l *Lead) Convert(customerID string) error {
	if l.Stage != Installation && l.Stage != Won {
		return errors.New("convert only from installation/won")
	}
	l.CustomerID = customerID
	l.Stage = Won
	l.UpdatedAt = time.Now().UTC()
	return nil
}
