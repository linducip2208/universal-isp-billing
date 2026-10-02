// Package lab implements the Connection Lab: test + capability discovery
// whose results are persisted for audit (devices/device_capabilities).
//
// Honesty rule: probes execute REAL connector methods with timeouts and
// record evidence per capability. A probe is marked ok ONLY when the
// underlying call succeeds — never from cached claims, and never from a
// mock when a real device was selected (mocks are not registered for real
// vendor families, so they cannot be selected here).
package lab

import (
	"context"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

// ConnectorVersion is the lab protocol version, shown in results.
const ConnectorVersion = "lab/2.0"

type ProbeResult struct {
	Capability sdk.Capability   `json:"capability"`
	Declared   sdk.Verification `json:"declared"`
	Observed   sdk.Verification `json:"observed"`
	LatencyMs  int64            `json:"latency_ms"`
	Error      string           `json:"error,omitempty"`
}

type TestResult struct {
	Vendor           string               `json:"vendor"`
	Model            string               `json:"model"`
	Firmware         string               `json:"firmware"`
	Version          string               `json:"version"`
	Serial           string               `json:"serial,omitempty"`
	UptimeSecs       int64                `json:"uptime_secs"`
	CPU              float64              `json:"cpu_percent"`
	Memory           float64              `json:"memory_percent"`
	LatencyMs        int64                `json:"latency_ms"`
	APIVersion       string               `json:"api_version,omitempty"`
	Interfaces       []sdk.Iface          `json:"interfaces"`
	Capabilities     []sdk.CapabilityInfo `json:"capabilities"`
	Probes           []ProbeResult        `json:"probes"`
	ConnectorVersion string               `json:"connector_version"`
	TestedAt         time.Time            `json:"tested_at"`
	Healthy          bool                 `json:"healthy"`
	Error            string               `json:"error,omitempty"`
}

func probe(ctx context.Context, c sdk.NetworkConnector, id sdk.Capability) ProbeResult {
	start := time.Now()
	pr := ProbeResult{Capability: id}
	pctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var err error
	switch id {
	case sdk.CapDeviceInfo, sdk.CapDeviceHealth:
		_, err = c.GetDeviceInfo(pctx)
	case sdk.CapInterfaces, sdk.CapIfaceMonitor:
		_, err = c.GetInterfaces(pctx)
	case sdk.CapClients, sdk.CapClientList, sdk.CapClientMonitor:
		_, err = c.GetClients(pctx)
	case sdk.CapTraffic:
		_, err = c.GetTraffic(pctx, "")
	case sdk.CapSiteDiscovery:
		_, err = c.GetSites(pctx)
	default:
		pr.Observed = sdk.Planned
		pr.Error = "no live probe for this capability (declaration only)"
		pr.LatencyMs = time.Since(start).Milliseconds()
		return pr
	}
	pr.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		pr.Observed = sdk.Partial
		pr.Error = err.Error()
		return pr
	}
	pr.Observed = sdk.Verified
	return pr
}

func TestAndDiscover(ctx context.Context, c sdk.NetworkConnector) (*TestResult, error) {
	start := time.Now()
	res := &TestResult{TestedAt: time.Now().UTC(), ConnectorVersion: ConnectorVersion}
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
		res.Capabilities = caps.Items
		for _, item := range caps.Items {
			pr := probe(ctx, c, item.ID)
			pr.Declared = item.Status
			// A failed probe never upgrades the declaration; a passed probe
			// confirms it for THIS device/model/firmware only.
			res.Probes = append(res.Probes, pr)
		}
	}
	res.LatencyMs = time.Since(start).Milliseconds()
	return res, nil
}
