// Package devices models inventory: sites, devices, polling cadence,
// capability snapshots. Secrets live in device_credentials (encrypted),
// never in this package's structs or API output.
package devices

import (
	"errors"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusOnline    Status = "online"
	StatusOffline   Status = "offline"
	StatusDegraded  Status = "degraded"
	StatusSimulated Status = "simulated"
)

type Device struct {
	ID             string             `json:"id"`
	OrgID          string             `json:"org_id"`
	SiteID         string             `json:"site_id,omitempty"`
	Vendor         string             `json:"vendor"`
	Family         string             `json:"family,omitempty"`
	Model          string             `json:"model,omitempty"`
	Firmware       string             `json:"firmware,omitempty"`
	Host           string             `json:"host"`
	ConnectionType sdk.ConnectionType `json:"connection_type"`
	Status         Status             `json:"status"`
	PollInterval   time.Duration      `json:"poll_interval"`
	Capabilities   map[string]string  `json:"capabilities,omitempty"`
	LastSuccess    *time.Time         `json:"last_success,omitempty"`
	LastError      string             `json:"last_error,omitempty"`
	LatencyMs      int64              `json:"latency_ms"`
}

func (d *Device) Validate() error {
	if d.Vendor == "" || d.Host == "" {
		return errors.New("vendor and host required")
	}
	if d.ConnectionType == "" {
		return errors.New("connection_type required")
	}
	if d.PollInterval != 0 && d.PollInterval < 10*time.Second {
		return errors.New("poll interval too aggressive (<10s)")
	}
	return nil
}

func DefaultPollInterval() time.Duration { return 60 * time.Second }
