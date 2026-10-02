// Package sdk defines the vendor-neutral NetworkConnector interface.
// Billing and provisioning code must depend ONLY on this interface,
// never on vendor-specific packages.
package sdk

import (
	"context"
	"time"
)

type ConnectionType string

const (
	ConnCloudAPI    ConnectionType = "cloud_api"
	ConnREST        ConnectionType = "rest"
	ConnRESTCONF    ConnectionType = "restconf"
	ConnNETCONF     ConnectionType = "netconf"
	ConnSSH         ConnectionType = "ssh"
	ConnCLI         ConnectionType = "cli"
	ConnRADIUS      ConnectionType = "radius"
	ConnSNMP        ConnectionType = "snmp"
	ConnWebhook     ConnectionType = "webhook"
	ConnGenericHTTP ConnectionType = "generic_http"
)

type Capability string

const (
	CapPPPoE         Capability = "pppoe"
	CapIPoE          Capability = "ipoe"
	CapHotspot       Capability = "hotspot"
	CapVoucher       Capability = "voucher"
	CapRADIUS        Capability = "radius"
	CapQueue         Capability = "queue"
	CapDHCP          Capability = "dhcp"
	CapFirewall      Capability = "firewall"
	CapIfaceMonitor  Capability = "interface_monitoring"
	CapTraffic       Capability = "traffic"
	CapDisconnect    Capability = "disconnect"
	CapCoA           Capability = "coa"
	CapProvisioning  Capability = "provisioning"
	CapVLAN          Capability = "vlan"
	CapOLT           Capability = "olt"
	CapONU           Capability = "onu"
	CapTR069         Capability = "tr069"
	CapClientList    Capability = "client_list"
	CapSiteDiscovery Capability = "site_discovery"
)

type CapabilityState string

const (
	StateVerified          CapabilityState = "VERIFIED"
	StateImplemented       CapabilityState = "IMPLEMENTED"
	StateModelDependent    CapabilityState = "MODEL_DEPENDENT"
	StateAPIRequired       CapabilityState = "API_REQUIRED"
	StateCredentialReq     CapabilityState = "CREDENTIAL_REQUIRED"
	StateNotSupported      CapabilityState = "NOT_SUPPORTED"
	StatePlanned           CapabilityState = "PLANNED"
	StateRequiresVendorAcc CapabilityState = "REQUIRES_VENDOR_ACCESS"
)

type DeviceInfo struct {
	Vendor        string  `json:"vendor"`
	Model         string  `json:"model"`
	Firmware      string  `json:"firmware"`
	Version       string  `json:"version"`
	Serial        string  `json:"serial,omitempty"`
	UptimeSecs    int64   `json:"uptime_secs"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	APIVersion    string  `json:"api_version,omitempty"`
}

type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Model    string `json:"model"`
	Status   string `json:"status"`
	Firmware string `json:"firmware,omitempty"`
}

type Client struct {
	MAC      string `json:"mac"`
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	AP       string `json:"ap,omitempty"`
	RSSI     int    `json:"rssi,omitempty"`
	RxBytes  int64  `json:"rx_bytes"`
	TxBytes  int64  `json:"tx_bytes"`
}

type Iface struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Running   bool    `json:"running"`
	RxBps     int64   `json:"rx_bps"`
	TxBps     int64   `json:"tx_bps"`
	RxPackets int64   `json:"rx_packets"`
	TxPackets int64   `json:"tx_packets"`
	Errors    int64   `json:"errors"`
	SpeedMbps int     `json:"speed_mbps"`
	Load      float64 `json:"load"`
}

type Traffic struct {
	Target    string    `json:"target"`
	RxBps     int64     `json:"rx_bps"`
	TxBps     int64     `json:"tx_bps"`
	SampledAt time.Time `json:"sampled_at"`
}

type Capabilities struct {
	States map[Capability]CapabilityState `json:"states"`
}

func (c *Capabilities) Has(cap Capability) bool {
	s, ok := c.States[cap]
	return ok && (s == StateVerified || s == StateImplemented)
}

type ProvisionRequest struct {
	SubscriberID   string            `json:"subscriber_id"`
	Username       string            `json:"username"`
	ServiceType    string            `json:"service_type"` // pppoe | ipoe | hotspot | ftth
	DownloadMbps   int               `json:"download_mbps"`
	UploadMbps     int               `json:"upload_mbps"`
	VLAN           int               `json:"vlan,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type UpdateRequest struct {
	SubscriberID   string            `json:"subscriber_id"`
	DownloadMbps   int               `json:"download_mbps"`
	UploadMbps     int               `json:"upload_mbps"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type Health struct {
	ConnectionType ConnectionType `json:"connection_type"`
	AuthType       string         `json:"auth_type"`
	Vendor         string         `json:"vendor"`
	Model          string         `json:"model"`
	Firmware       string         `json:"firmware"`
	Healthy        bool           `json:"healthy"`
	LatencyMs      int64          `json:"latency_ms"`
	LastSuccess    *time.Time     `json:"last_success,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
}

// NetworkConnector is THE abstraction boundary. All vendor packages implement it.
type NetworkConnector interface {
	ConnectionType() ConnectionType
	TestConnection(ctx context.Context) error
	GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)
	GetCapabilities(ctx context.Context) (*Capabilities, error)
	GetSites(ctx context.Context) ([]Site, error)
	GetDevices(ctx context.Context) ([]Device, error)
	GetClients(ctx context.Context) ([]Client, error)
	GetInterfaces(ctx context.Context) ([]Iface, error)
	GetTraffic(ctx context.Context, target string) (*Traffic, error)
	ProvisionSubscriber(ctx context.Context, req ProvisionRequest) error
	UpdateSubscriber(ctx context.Context, req UpdateRequest) error
	SuspendSubscriber(ctx context.Context, subscriberID string) error
	ActivateSubscriber(ctx context.Context, subscriberID string) error
	DisconnectSubscriber(ctx context.Context, subscriberID string) error
	DeleteSubscriber(ctx context.Context, subscriberID string) error
	Health() Health
}
