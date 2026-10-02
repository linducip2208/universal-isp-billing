// Package field: tickets, work orders, technicians, scheduling, SLA hooks,
// spare parts. Linked to customer/subscriber/device/incident by IDs.
package field

import (
	"errors"
	"time"
)

type TicketStatus string

const (
	TicketOpen       TicketStatus = "open"
	TicketAssigned   TicketStatus = "assigned"
	TicketInProgress TicketStatus = "in_progress"
	TicketResolved   TicketStatus = "resolved"
	TicketClosed     TicketStatus = "closed"
)

type Ticket struct {
	ID           string       `json:"id"`
	OrgID        string       `json:"org_id"`
	CustomerID   string       `json:"customer_id,omitempty"`
	SubscriberID string       `json:"subscriber_id,omitempty"`
	DeviceID     string       `json:"device_id,omitempty"`
	IncidentID   string       `json:"incident_id,omitempty"`
	Subject      string       `json:"subject"`
	Status       TicketStatus `json:"status"`
	Priority     string       `json:"priority"`
	CreatedAt    time.Time    `json:"created_at"`
}

type WorkOrder struct {
	ID           string       `json:"id"`
	OrgID        string       `json:"org_id"`
	TicketID     string       `json:"ticket_id,omitempty"`
	TechnicianID string       `json:"technician_id,omitempty"`
	Kind         string       `json:"kind"` // install | maintenance | replacement | survey
	Status       TicketStatus `json:"status"`
	ScheduledFor *time.Time   `json:"scheduled_for,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	PhotoRefs    []string     `json:"photo_refs,omitempty"`
	SignatureRef string       `json:"signature_ref,omitempty"`
}

type Technician struct {
	ID    string `json:"id"`
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
	Area  string `json:"area,omitempty"`
}

type SparePart struct {
	ID       string `json:"id"`
	OrgID    string `json:"org_id"`
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// Advance enforces the ticket state machine.
func Advance(t *Ticket, to TicketStatus) error {
	ok := map[TicketStatus][]TicketStatus{
		TicketOpen:       {TicketAssigned, TicketClosed},
		TicketAssigned:   {TicketInProgress, TicketOpen},
		TicketInProgress: {TicketResolved, TicketAssigned},
		TicketResolved:   {TicketClosed, TicketInProgress},
		TicketClosed:     {},
	}[t.Status]
	for _, s := range ok {
		if s == to {
			t.Status = to
			return nil
		}
	}
	return errors.New("illegal ticket transition")
}
