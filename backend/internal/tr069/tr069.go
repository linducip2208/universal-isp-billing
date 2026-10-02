// Package tr069 defines the ACS (Auto Configuration Server) architecture
// seam for CPE management (Inform, Get/SetParameterValues, Reboot, Firmware
// upgrade). The full CWMP session server is roadmap; this package fixes the
// interfaces so provisioning and FTTH code can depend on them today.
package tr069

import (
	"context"
	"time"
)

// DeviceID mirrors CWMP DeviceIdStruct.
type DeviceID struct {
	Manufacturer string `json:"manufacturer"`
	OUI          string `json:"oui"`
	ProductClass string `json:"product_class"`
	SerialNumber string `json:"serial_number"`
}

// Inform is a CPE bootstrap/session-start event.
type Inform struct {
	Device   DeviceID          `json:"device"`
	Event    string            `json:"event"` // BOOT | PERIODIC | VALUE_CHANGE
	Params   map[string]string `json:"params,omitempty"`
	Received time.Time         `json:"received"`
}

// Task is a queued ACS action for one CPE.
type Task struct {
	Serial string            `json:"serial"`
	Action string            `json:"action"` // GetParameterValues | SetParameterValues | Reboot | Upgrade
	Params map[string]string `json:"params,omitempty"`
}

// ACS is the interface the future CWMP server (and today's mocks) implement.
type ACS interface {
	IngestInform(ctx context.Context, in Inform) error
	EnqueueTask(ctx context.Context, t Task) error
	PendingTasks(ctx context.Context, serial string) ([]Task, error)
	AckTask(ctx context.Context, serial, action string) error
}
