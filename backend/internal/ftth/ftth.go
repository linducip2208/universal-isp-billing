// Package ftth: vendor-neutral FTTH domain — OLT/shelf/slot/PON/splitter/ONU,
// optical thresholds, profiles, authorize/provision/deprovision plans and
// diagnostics classification. Vendor adapters implement VendorAdapter; the
// registry holds 13 PLANNED families (fail-closed) — see docs.
package ftth

import "fmt"

type OLT struct {
	ID       string `json:"id"`
	Vendor   string `json:"vendor"`
	Model    string `json:"model"`
	Shelf    int    `json:"shelf"`
	Slot     int    `json:"slot"`
	PONCount int    `json:"pon_count"`
}

type PON struct {
	OLTID string `json:"olt_id"`
	Port  string `json:"port"` // e.g. "1/1/1"
}

type Splitter struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Ratio string `json:"ratio"` // e.g. "1:16"
}

type ONU struct {
	ID             string  `json:"id"`
	Serial         string  `json:"serial"`
	LOID           string  `json:"loid,omitempty"`
	Status         string  `json:"status"` // online | offline | dyinggasp | unknown
	RxPower        float64 `json:"rx_power_dbm"`
	TxPower        float64 `json:"tx_power_dbm"`
	Temperature    float64 `json:"temperature_c"`
	DistanceM      int     `json:"distance_m"`
	LineProfile    string  `json:"line_profile,omitempty"`
	ServiceProfile string  `json:"service_profile,omitempty"`
	VLAN           int     `json:"vlan"`
}

type Profile struct {
	Vendor   string `json:"vendor"`
	Name     string `json:"name"`
	DownMbps int    `json:"down_mbps"`
	UpMbps   int    `json:"up_mbps"`
	VLAN     int    `json:"vlan"`
	Mode     string `json:"mode"` // bridge | router
}

// Optical thresholds (GPON class B+ typical): warn below -25, critical below -28.
const (
	RxWarnDbm = -25.0
	RxCritDbm = -28.0
)

type Diagnosis struct {
	Level   string   `json:"level"` // ok | degraded | down | unknown
	Reasons []string `json:"reasons"`
}

// Diagnose classifies ONU health from telemetry. Never invents causes —
// every reason cites the measured field.
func Diagnose(o ONU) Diagnosis {
	d := Diagnosis{Level: "ok"}
	add := func(level, reason string) {
		d.Reasons = append(d.Reasons, reason)
		if level == "down" {
			d.Level = "down"
		} else if d.Level == "ok" && level == "degraded" {
			d.Level = "degraded"
		}
	}
	switch o.Status {
	case "online":
	case "offline":
		add("down", "onu admin status offline")
	case "dyinggasp":
		add("down", "onu dying-gasp (power loss likely)")
	default:
		add("degraded", fmt.Sprintf("onu status unknown (%q)", o.Status))
	}
	if o.RxPower != 0 {
		switch {
		case o.RxPower < RxCritDbm:
			add("down", fmt.Sprintf("rx %.1fdBm below critical %.0f", o.RxPower, RxCritDbm))
		case o.RxPower < RxWarnDbm:
			add("degraded", fmt.Sprintf("rx %.1fdBm below warning %.0f", o.RxPower, RxWarnDbm))
		}
	}
	if o.Temperature > 75 {
		add("degraded", fmt.Sprintf("temperature %.0fC high", o.Temperature))
	}
	if len(d.Reasons) == 0 {
		d.Reasons = []string{"all telemetry nominal"}
	}
	return d
}

// Plan is a vendor-neutral provisioning intent executed by an adapter.
type Plan struct {
	Action  string            `json:"action"` // authorize|provision|deprovision|restart
	Serial  string            `json:"serial"`
	Profile Profile           `json:"profile,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
}

func Authorize(serial string, p Profile) Plan {
	return Plan{Action: "authorize", Serial: serial, Profile: p}
}

func Provision(serial string, p Profile, vlan int) Plan {
	p.VLAN = vlan
	return Plan{Action: "provision", Serial: serial, Profile: p,
		Params: map[string]string{"vlan": fmt.Sprint(vlan)}}
}

func Deprovision(serial string) Plan { return Plan{Action: "deprovision", Serial: serial} }
func Restart(serial string) Plan     { return Plan{Action: "restart", Serial: serial} }

// VendorAdapter executes Plans on real OLTs. No adapter claims support
// without hardware tests — all 13 families are PLANNED/fail-closed.
type VendorAdapter interface {
	Vendor() string
	Execute(plan Plan) error
}
