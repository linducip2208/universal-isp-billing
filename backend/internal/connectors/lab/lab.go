// Package lab implements the Connection Lab: test + capability discovery
// whose results are persisted for audit (devices/device_capabilities).
package lab

import (
	"context"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type TestResult struct {
	Vendor       string                                 `json:"vendor"`
	Model        string                                 `json:"model"`
	Firmware     string                                 `json:"firmware"`
	Version      string                                 `json:"version"`
	Serial       string                                 `json:"serial,omitempty"`
	UptimeSecs   int64                                  `json:"uptime_secs"`
	CPU          float64                                `json:"cpu_percent"`
	Memory       float64                                `json:"memory_percent"`
	LatencyMs    int64                                  `json:"latency_ms"`
	APIVersion   string                                 `json:"api_version,omitempty"`
	Interfaces   []sdk.Iface                            `json:"interfaces"`
	Capabilities map[sdk.Capability]sdk.CapabilityState `json:"capabilities"`
	TestedAt     time.Time                              `json:"tested_at"`
	Healthy      bool                                   `json:"healthy"`
	Error        string                                 `json:"error,omitempty"`
}

func TestAndDiscover(ctx context.Context, c sdk.NetworkConnector) (*TestResult, error) {
	start := time.Now()
	res := &TestResult{TestedAt: time.Now().UTC()}
	if err := c.TestConnection(ctx); err != nil {
		res.Healthy = false
		res.Error = err.Error()
		res.LatencyMs = time.Since(start).Milliseconds()
		return res, err
	}
	res.Healthy = true
	if info, err := c.GetDeviceInfo(ctx); err == nil && info != nil {
		res.Vendor, res.Model, res.Firmware = info.Vendor, info.Model, info.Firmware
		res.Version, res.Serial = info.Version, info.Serial
		res.UptimeSecs, res.CPU, res.Memory = info.UptimeSecs, info.CPUPercent, info.MemoryPercent
		res.APIVersion = info.APIVersion
	}
	if ifaces, err := c.GetInterfaces(ctx); err == nil {
		res.Interfaces = ifaces
	}
	if caps, err := c.GetCapabilities(ctx); err == nil && caps != nil {
		res.Capabilities = caps.States
	}
	res.LatencyMs = time.Since(start).Milliseconds()
	h := c.Health()
	if h.LatencyMs == 0 {
		h.LatencyMs = res.LatencyMs
	}
	return res, nil
}
