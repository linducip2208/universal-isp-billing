// Package sdk defines the vendor-neutral NetworkConnector interface.
// Billing and provisioning code must depend ONLY on this interface,
// never on vendor-specific packages.
//
// Capability model 2.0: granular capability IDs, per-capability versions
// and firmware notes, and a strict 6-state verification model:
//
//	VERIFIED, PARTIAL, READY_FOR_CREDENTIALS,
//	REQUIRES_VENDOR_ACCESS, UNSUPPORTED, PLANNED
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

// Capability is a granular, versionable operation/feature identifier.
type Capability string

const (
	CapDeviceInfo    Capability = "device_info"
	CapDeviceHealth  Capability = "device_health"
	CapInterfaces    Capability = "interfaces"
	CapTraffic       Capability = "traffic"
	CapClients       Capability = "clients"
	CapClientMonitor Capability = "client_monitoring"
	CapUsers         Capability = "users"
	CapSubscribers   Capability = "subscribers"
	CapPPPoE         Capability = "pppoe"
	CapIPoE          Capability = "ipoe"
	CapHotspot       Capability = "hotspot"
	CapVoucher       Capability = "voucher"
	CapDHCP          Capability = "dhcp"
	CapIPPools       Capability = "ip_pools"
	CapVLAN          Capability = "vlan"
	CapFirewall      Capability = "firewall"
	CapNAT           Capability = "nat"
	CapRoutes        Capability = "routes"
	CapQueues        Capability = "queues"
	CapBandwidth     Capability = "bandwidth"
	CapQoS           Capability = "qos"
	CapRADIUS        Capability = "radius"
	CapCoA           Capability = "coa"
	CapDisconnect    Capability = "disconnect"
	CapProvisioning  Capability = "provisioning"
	CapSuspend       Capability = "suspend"
	CapActivate      Capability = "activate"
	CapDelete        Capability = "delete"
	CapReboot        Capability = "reboot"
	CapConfigBackup  Capability = "config_backup"
	CapConfigRestore Capability = "config_restore"
	CapWireless      Capability = "wireless"
	CapAP            Capability = "ap"
	CapSSID          Capability = "ssid"
	CapOLT           Capability = "olt"
	CapONU           Capability = "onu"
	CapOpticalPower  Capability = "optical_power"
	CapTR069         Capability = "tr069"
	CapSyslog        Capability = "syslog"
	CapSNMP          Capability = "snmp"
	CapNETCONF       Capability = "netconf"
	CapRESTCONF      Capability = "restconf"
	CapSSH           Capability = "ssh"
	CapCLI           Capability = "cli"
	CapWebhooks      Capability = "webhooks"
	CapSiteDiscovery Capability = "site_discovery"
	CapIfaceMonitor  Capability = "interface_monitoring"
	CapQueue         Capability = "queue" // legacy alias of queues
	CapClientList    Capability = "client_list"
)

// Verification is the strict connector/capability status model (12 states).
// Promotion rule: automated evidence first (AUTOMATED_TESTED), then lab runs
// (LAB_TESTED), then real devices (REAL_DEVICE_VERIFIED), then production
// hardening (PRODUCTION_READY). Nothing here is above AUTOMATED_TESTED today.
type Verification string

const (
	ArchitectureReady    Verification = "ARCHITECTURE_READY"
	Implemented          Verification = "IMPLEMENTED"
	ProtocolImplemented  Verification = "PROTOCOL_IMPLEMENTED"
	AutomatedTested      Verification = "AUTOMATED_TESTED"
	LabTested            Verification = "LAB_TESTED"
	RealDeviceVerified   Verification = "REAL_DEVICE_VERIFIED"
	ProductionReady      Verification = "PRODUCTION_READY"
	Verified             Verification = "VERIFIED" // legacy alias of REAL_DEVICE_VERIFIED
	Partial              Verification = "PARTIAL"
	ReadyForCredentials  Verification = "READY_FOR_CREDENTIALS"
	RequiresVendorAccess Verification = "REQUIRES_VENDOR_ACCESS"
	Unsupported          Verification = "UNSUPPORTED"
	Planned              Verification = "PLANNED"
)

// CapabilityState is kept as an alias so older call sites keep compiling;
// new code must use Verification.
type CapabilityState = Verification

const (
	StateVerified          = Verified
	StateImplemented       = Partial
	StateModelDependent    = Partial
	StateAPIRequired       = ReadyForCredentials
	StateCredentialReq     = ReadyForCredentials
	StateNotSupported      = Unsupported
	StatePlanned           = Planned
	StateRequiresVendorAcc = RequiresVendorAccess
)

// CapabilityInfo describes one granular capability with version/firmware
// awareness and model-specific notes.
type CapabilityInfo struct {
	ID          Capability   `json:"id"`
	Status      Verification `json:"status"`
	Version     string       `json:"version,omitempty"`
	MinFirmware string       `json:"min_firmware,omitempty"`
	Models      []string     `json:"models,omitempty"`
	Note        string       `json:"note,omitempty"`
}

type Capabilities struct {
	Items []CapabilityInfo `json:"items"`
}

func NewCapabilities(items ...CapabilityInfo) *Capabilities {
	return &Capabilities{Items: items}
}

// Cap is a helper for the common case: implemented/partial without notes.
func Cap(id Capability, st Verification) CapabilityInfo {
	return CapabilityInfo{ID: id, Status: st}
}

func (c *Capabilities) StatusOf(id Capability) (Verification, bool) {
	if c == nil {
		return Unsupported, false
	}
	for _, it := range c.Items {
		if it.ID == id {
			return it.Status, true
		}
	}
	return Unsupported, false
}

// Has reports whether the capability is usable (verified or partial).
func (c *Capabilities) Has(id Capability) bool {
	st, ok := c.StatusOf(id)
	return ok && (st == Verified || st == Partial)
}

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

// Descriptor is the full connector identity card served by the registry,
// Connection Lab, and docs/capability-matrix.yaml.
type Descriptor struct {
	Vendor           string           `json:"vendor"`
	ProductFamily    string           `json:"product_family"`
	Models           []string         `json:"models,omitempty"`
	ConnectionType   ConnectionType   `json:"connection_type"`
	Protocols        []string         `json:"protocols,omitempty"`
	AuthMethods      []string         `json:"auth_methods,omitempty"`
	Capabilities     []CapabilityInfo `json:"capabilities,omitempty"`
	Limitations      []string         `json:"limitations,omitempty"`
	ConnectorVersion string           `json:"connector_version"`
	Status           Verification     `json:"status"`
	DocURL           string           `json:"doc_url,omitempty"`
	VerifiedAt       string           `json:"verified_at,omitempty"`
	Evidence         string           `json:"evidence,omitempty"`
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
