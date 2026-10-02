package mikrotik

import (
	"context"
	"strconv"
	"strings"
)

// Production RouterOS operations. Every method performs real API I/O using
// documented RouterOS paths (/ppp/secret, /ppp/active, /queue/simple,
// /ip/dhcp-server/lease, /ip/hotspot/user, /ip/pool, /ppp/profile,
// /system/identity, /system/resource, /system/reboot, /export).

type PPPSecret struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Profile  string `json:"profile"`
	Disabled bool   `json:"disabled"`
	Comment  string `json:"comment,omitempty"`
}

type PPPActive struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Uptime  string `json:"uptime"`
}

type DHCPLease struct {
	ID      string `json:"id"`
	MAC     string `json:"mac"`
	Address string `json:"address"`
	Host    string `json:"host,omitempty"`
	Status  string `json:"status,omitempty"`
}

type HotspotUser struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Profile string `json:"profile"`
	UpTime  string `json:"uptime,omitempty"`
}

type IPPool struct {
	Name   string `json:"name"`
	Ranges string `json:"ranges"`
}

type PPPProfile struct {
	Name      string `json:"name"`
	LocalAddr string `json:"local_address,omitempty"`
	RateLimit string `json:"rate_limit,omitempty"`
}

type QueueEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Target   string `json:"target"`
	MaxLimit string `json:"max_limit"`
	Disabled bool   `json:"disabled"`
}

func (c *Connector) GetPPPSecrets(ctx context.Context) ([]PPPSecret, error) {
	rows, err := c.apiCall(ctx, "/ppp/secret/print")
	if err != nil {
		return nil, err
	}
	var out []PPPSecret
	for _, r := range rows {
		out = append(out, PPPSecret{ID: r[".id"], Name: r["name"], Profile: r["profile"], Disabled: r["disabled"] == "true", Comment: r["comment"]})
	}
	return out, nil
}

func (c *Connector) GetPPPActive(ctx context.Context) ([]PPPActive, error) {
	rows, err := c.apiCall(ctx, "/ppp/active/print")
	if err != nil {
		return nil, err
	}
	var out []PPPActive
	for _, r := range rows {
		out = append(out, PPPActive{ID: r[".id"], Name: r["name"], Address: r["address"], Uptime: r["uptime"]})
	}
	return out, nil
}

func (c *Connector) GetDHCPLeases(ctx context.Context) ([]DHCPLease, error) {
	rows, err := c.apiCall(ctx, "/ip/dhcp-server/lease/print")
	if err != nil {
		return nil, err
	}
	var out []DHCPLease
	for _, r := range rows {
		out = append(out, DHCPLease{ID: r[".id"], MAC: r["mac-address"], Address: r["address"], Host: r["host-name"], Status: r["status"]})
	}
	return out, nil
}

func (c *Connector) GetHotspotUsers(ctx context.Context) ([]HotspotUser, error) {
	rows, err := c.apiCall(ctx, "/ip/hotspot/user/print")
	if err != nil {
		return nil, err
	}
	var out []HotspotUser
	for _, r := range rows {
		out = append(out, HotspotUser{ID: r[".id"], Name: r["name"], Profile: r["profile"], UpTime: r["uptime"]})
	}
	return out, nil
}

func (c *Connector) GetIPPools(ctx context.Context) ([]IPPool, error) {
	rows, err := c.apiCall(ctx, "/ip/pool/print")
	if err != nil {
		return nil, err
	}
	var out []IPPool
	for _, r := range rows {
		out = append(out, IPPool{Name: r["name"], Ranges: r["ranges"]})
	}
	return out, nil
}

func (c *Connector) GetPPPProfiles(ctx context.Context) ([]PPPProfile, error) {
	rows, err := c.apiCall(ctx, "/ppp/profile/print")
	if err != nil {
		return nil, err
	}
	var out []PPPProfile
	for _, r := range rows {
		out = append(out, PPPProfile{Name: r["name"], LocalAddr: r["local-address"], RateLimit: r["rate-limit"]})
	}
	return out, nil
}

func (c *Connector) GetQueues(ctx context.Context) ([]QueueEntry, error) {
	rows, err := c.apiCall(ctx, "/queue/simple/print")
	if err != nil {
		return nil, err
	}
	var out []QueueEntry
	for _, r := range rows {
		out = append(out, QueueEntry{ID: r[".id"], Name: r["name"], Target: r["target"], MaxLimit: r["max-limit"], Disabled: r["disabled"] == "true"})
	}
	return out, nil
}

func (c *Connector) GetIdentity(ctx context.Context) (string, error) {
	rows, err := c.apiCall(ctx, "/system/identity/print")
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	return rows[0]["name"], nil
}

// EnsurePPPProfile creates a rate-limited PPP profile for a package tier.
// Idempotent: existing profile with the same name is left untouched.
func (c *Connector) EnsurePPPProfile(ctx context.Context, name string, downMbps, upMbps int) error {
	existing, err := c.apiCall(ctx, "/ppp/profile/print", "?name="+name)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	rate := strconv.Itoa(upMbps) + "M/" + strconv.Itoa(downMbps) + "M"
	_, err = c.apiCall(ctx, "/ppp/profile/add", "=name="+name, "=rate-limit="+rate)
	return err
}

// ExportConfig runs /export and returns the RouterOS script (config backup).
func (c *Connector) ExportConfig(ctx context.Context) (string, error) {
	rows, err := c.apiCall(ctx, "/export")
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, r := range rows {
		for k, v := range r {
			sb.WriteString(k + "=" + v + "\n")
		}
	}
	return sb.String(), nil
}

// Reboot issues /system/reboot (destructive — callers must audit + confirm).
func (c *Connector) Reboot(ctx context.Context) error {
	_, err := c.apiCall(ctx, "/system/reboot")
	return err
}
