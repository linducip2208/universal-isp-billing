// Desired-state vs actual-state drift detection for provisioning.
// The scheduler compares what billing sold (Desired) with what the device
// enforces (Actual); differences become explicit Plans an operator or
// automation can apply, verify, or roll back.
package provisioning

import (
	"context"
	"fmt"
	"strings"
)

type Desired struct {
	SubscriptionID string `json:"subscription_id"`
	Username       string `json:"username"`
	DownloadMbps   int    `json:"download_mbps"`
	UploadMbps     int    `json:"upload_mbps"`
	VLAN           int    `json:"vlan,omitempty"`
	Service        string `json:"service"`
}

type Actual struct {
	Found        bool   `json:"found"`
	DownloadMbps int    `json:"download_mbps"`
	UploadMbps   int    `json:"upload_mbps"`
	VLAN         int    `json:"vlan,omitempty"`
	Suspended    bool   `json:"suspended"`
	Raw          string `json:"raw,omitempty"`
}

type DriftAction string

const (
	ActionCreate   DriftAction = "create"
	ActionUpdate   DriftAction = "update"
	ActionSuspend  DriftAction = "suspend"
	ActionActivate DriftAction = "activate"
	ActionNone     DriftAction = "none"
)

type Plan struct {
	Action DriftAction `json:"action"`
	Reason string      `json:"reason"`
}

// Detect compares desired vs actual and returns the minimal plan.
// No network I/O here — pure decision logic, fully testable.
func Detect(want Desired, got Actual, wantSuspended bool) Plan {
	if !got.Found {
		return Plan{Action: ActionCreate, Reason: "subscriber missing on device"}
	}
	if wantSuspended && !got.Suspended {
		return Plan{Action: ActionSuspend, Reason: "billing state suspended, device still active"}
	}
	if !wantSuspended && got.Suspended {
		return Plan{Action: ActionActivate, Reason: "billing state active, device suspended"}
	}
	var diffs []string
	if want.DownloadMbps != got.DownloadMbps {
		diffs = append(diffs, fmt.Sprintf("down %d!=%d", want.DownloadMbps, got.DownloadMbps))
	}
	if want.UploadMbps != got.UploadMbps {
		diffs = append(diffs, fmt.Sprintf("up %d!=%d", want.UploadMbps, got.UploadMbps))
	}
	if want.VLAN != got.VLAN {
		diffs = append(diffs, fmt.Sprintf("vlan %d!=%d", want.VLAN, got.VLAN))
	}
	if len(diffs) > 0 {
		return Plan{Action: ActionUpdate, Reason: "bandwidth drift: " + strings.Join(diffs, ", ")}
	}
	return Plan{Action: ActionNone, Reason: "in sync"}
}

// ActualFetcher retrieves device truth for one subscriber.
type ActualFetcher func(ctx context.Context, subscriptionID string) (Actual, error)
