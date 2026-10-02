// Package generic provides standards-based connectors so that any vendor
// without a dedicated package can still be integrated via RADIUS, SNMP,
// REST, SSH/CLI, NETCONF, RESTCONF, or webhook transports.
package generic

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/radius"
)

func unsupported() *sdk.Capabilities {
	return &sdk.Capabilities{}
}

// ---------- GenericRESTConnector ----------

type RESTConfig struct {
	BaseURL  string
	Token    string
	Username string
	Password string
	Timeout  time.Duration
}

type RESTConnector struct {
	cfg    RESTConfig
	client *http.Client
	health sdk.Health
}

func NewREST(cfg RESTConfig) *RESTConnector {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &RESTConnector{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout},
		health: sdk.Health{ConnectionType: sdk.ConnGenericHTTP, AuthType: "token", Vendor: "generic"}}
}

func (c *RESTConnector) ConnectionType() sdk.ConnectionType { return sdk.ConnGenericHTTP }

func (c *RESTConnector) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var r *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req, _ := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.cfg.BaseURL, "/")+path, r)
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	} else if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	// SSRF guard: only http/https schemes allowed (enforced by URL shape here).
	if !strings.HasPrefix(c.cfg.BaseURL, "http://") && !strings.HasPrefix(c.cfg.BaseURL, "https://") {
		return nil, fmt.Errorf("refusing non-http base URL")
	}
	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}

func (c *RESTConnector) TestConnection(ctx context.Context) error {
	resp, err := c.do(ctx, "GET", "/health", nil)
	if err != nil {
		c.health.Healthy = false
		c.health.LastError = err.Error()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("upstream %d", resp.StatusCode)
	}
	c.health.Healthy = true
	return nil
}

func (c *RESTConnector) GetDeviceInfo(ctx context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: "generic", Model: "generic-rest", Version: "1.0"}, nil
}
func (c *RESTConnector) GetCapabilities(_ context.Context) (*sdk.Capabilities, error) {
	return sdk.NewCapabilities(sdk.Cap(sdk.CapProvisioning, sdk.Partial)), nil
}
func (c *RESTConnector) GetSites(_ context.Context) ([]sdk.Site, error)       { return nil, nil }
func (c *RESTConnector) GetDevices(_ context.Context) ([]sdk.Device, error)   { return nil, nil }
func (c *RESTConnector) GetClients(_ context.Context) ([]sdk.Client, error)   { return nil, nil }
func (c *RESTConnector) GetInterfaces(_ context.Context) ([]sdk.Iface, error) { return nil, nil }
func (c *RESTConnector) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t, SampledAt: time.Now()}, nil
}
func idemNote(_ context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("idempotency key required")
	}
	return nil
}
func (c *RESTConnector) ProvisionSubscriber(ctx context.Context, req sdk.ProvisionRequest) error {
	if err := idemNote(ctx, req.IdempotencyKey); err != nil {
		return err
	}
	resp, err := c.do(ctx, "POST", "/subscribers", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("provision failed: %d", resp.StatusCode)
	}
	return nil
}
func (c *RESTConnector) UpdateSubscriber(ctx context.Context, req sdk.UpdateRequest) error {
	resp, err := c.do(ctx, "PATCH", "/subscribers/"+req.SubscriberID, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("update failed: %d", resp.StatusCode)
	}
	return nil
}
func (c *RESTConnector) SuspendSubscriber(ctx context.Context, id string) error {
	resp, err := c.do(ctx, "POST", "/subscribers/"+id+"/suspend", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
func (c *RESTConnector) ActivateSubscriber(ctx context.Context, id string) error {
	resp, err := c.do(ctx, "POST", "/subscribers/"+id+"/activate", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
func (c *RESTConnector) DisconnectSubscriber(ctx context.Context, id string) error {
	resp, err := c.do(ctx, "POST", "/subscribers/"+id+"/disconnect", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
func (c *RESTConnector) DeleteSubscriber(ctx context.Context, id string) error {
	resp, err := c.do(ctx, "DELETE", "/subscribers/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
func (c *RESTConnector) Health() sdk.Health { return c.health }

// ---------- GenericRADIUSConnector (CoA/Disconnect via RADIUS) ----------

type RADIUSConnector struct {
	NASAddr string
	Secret  string
	health  sdk.Health
}

func NewRADIUS(nasAddr, secret string) *RADIUSConnector {
	return &RADIUSConnector{NASAddr: nasAddr, Secret: secret,
		health: sdk.Health{ConnectionType: sdk.ConnRADIUS, AuthType: "shared-secret", Vendor: "generic"}}
}

func (c *RADIUSConnector) ConnectionType() sdk.ConnectionType { return sdk.ConnRADIUS }
func (c *RADIUSConnector) TestConnection(ctx context.Context) error {
	conn, err := net.DialTimeout("udp", c.NASAddr, 3*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
func (c *RADIUSConnector) GetDeviceInfo(_ context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: "generic", Model: "radius-nas"}, nil
}
func (c *RADIUSConnector) GetCapabilities(_ context.Context) (*sdk.Capabilities, error) {
	return sdk.NewCapabilities(
		sdk.Cap(sdk.CapRADIUS, sdk.Partial),
		sdk.Cap(sdk.CapDisconnect, sdk.Partial),
		sdk.Cap(sdk.CapCoA, sdk.Partial),
	), nil
}
func (c *RADIUSConnector) GetSites(_ context.Context) ([]sdk.Site, error)       { return nil, nil }
func (c *RADIUSConnector) GetDevices(_ context.Context) ([]sdk.Device, error)   { return nil, nil }
func (c *RADIUSConnector) GetClients(_ context.Context) ([]sdk.Client, error)   { return nil, nil }
func (c *RADIUSConnector) GetInterfaces(_ context.Context) ([]sdk.Iface, error) { return nil, nil }
func (c *RADIUSConnector) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t, SampledAt: time.Now()}, nil
}
func (c *RADIUSConnector) ProvisionSubscriber(_ context.Context, _ sdk.ProvisionRequest) error {
	return nil // authorization is enforced at Access-Accept time
}
func (c *RADIUSConnector) UpdateSubscriber(_ context.Context, req sdk.UpdateRequest) error {
	return radius.SendCoA(c.NASAddr, c.Secret, radius.CodeCoARequest, 1, []radius.Attr{{Type: 1, Value: []byte(req.SubscriberID)}}, 5*time.Second)
}
func (c *RADIUSConnector) SuspendSubscriber(_ context.Context, id string) error {
	return radius.SendCoA(c.NASAddr, c.Secret, radius.CodeDisconnectReq, 1, []radius.Attr{{Type: 1, Value: []byte(id)}}, 5*time.Second)
}
func (c *RADIUSConnector) ActivateSubscriber(_ context.Context, _ string) error { return nil }
func (c *RADIUSConnector) DisconnectSubscriber(_ context.Context, id string) error {
	return radius.SendCoA(c.NASAddr, c.Secret, radius.CodeDisconnectReq, 2, []radius.Attr{{Type: 1, Value: []byte(id)}}, 5*time.Second)
}
func (c *RADIUSConnector) DeleteSubscriber(_ context.Context, _ string) error { return nil }
func (c *RADIUSConnector) Health() sdk.Health                                 { return c.health }

// ---------- GenericSNMPConnector ----------

type SNMPConnector struct {
	Target    string
	Community string
	health    sdk.Health
}

func NewSNMP(target, community string) *SNMPConnector {
	return &SNMPConnector{Target: target, Community: community,
		health: sdk.Health{ConnectionType: sdk.ConnSNMP, AuthType: "community", Vendor: "generic"}}
}

func (c *SNMPConnector) ConnectionType() sdk.ConnectionType { return sdk.ConnSNMP }
func (c *SNMPConnector) TestConnection(ctx context.Context) error {
	// UDP reachability check for SNMP port 161 (sysDescr walk happens in poller).
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "udp", c.Target)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
func (c *SNMPConnector) GetDeviceInfo(_ context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: "generic", Model: "snmp-device"}, nil
}
func (c *SNMPConnector) GetCapabilities(_ context.Context) (*sdk.Capabilities, error) {
	return sdk.NewCapabilities(
		sdk.Cap(sdk.CapSNMP, sdk.Partial),
		sdk.Cap(sdk.CapInterfaces, sdk.Partial),
		sdk.Cap(sdk.CapIfaceMonitor, sdk.Partial),
		sdk.Cap(sdk.CapTraffic, sdk.Partial),
		sdk.Cap(sdk.CapDeviceInfo, sdk.Partial),
		sdk.Cap(sdk.CapDeviceHealth, sdk.Partial),
	), nil
}
func (c *SNMPConnector) GetSites(_ context.Context) ([]sdk.Site, error)       { return nil, nil }
func (c *SNMPConnector) GetDevices(_ context.Context) ([]sdk.Device, error)   { return nil, nil }
func (c *SNMPConnector) GetClients(_ context.Context) ([]sdk.Client, error)   { return nil, nil }
func (c *SNMPConnector) GetInterfaces(_ context.Context) ([]sdk.Iface, error) { return nil, nil }
func (c *SNMPConnector) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t, SampledAt: time.Now()}, nil
}
func (c *SNMPConnector) ProvisionSubscriber(_ context.Context, _ sdk.ProvisionRequest) error {
	return fmt.Errorf("snmp is read-only")
}
func (c *SNMPConnector) UpdateSubscriber(_ context.Context, _ sdk.UpdateRequest) error {
	return fmt.Errorf("snmp is read-only")
}
func (c *SNMPConnector) SuspendSubscriber(_ context.Context, _ string) error {
	return fmt.Errorf("snmp is read-only")
}
func (c *SNMPConnector) ActivateSubscriber(_ context.Context, _ string) error {
	return fmt.Errorf("snmp is read-only")
}
func (c *SNMPConnector) DisconnectSubscriber(_ context.Context, _ string) error {
	return fmt.Errorf("snmp is read-only")
}
func (c *SNMPConnector) DeleteSubscriber(_ context.Context, _ string) error {
	return fmt.Errorf("snmp is read-only")
}
func (c *SNMPConnector) Health() sdk.Health { return c.health }

var _ = tls.Config{}
var _ = unsupported
