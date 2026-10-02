// Package entitlements: commercial plan limits (Community, Professional,
// Business, Enterprise). Pure evaluation: usage vs plan returns violations
// for admin display/API — enforcement points consume this, nothing is
// artificially disabled in engineering paths.
package entitlements

type Plan string

const (
	Community    Plan = "community"
	Professional Plan = "professional"
	Business     Plan = "business"
	Enterprise   Plan = "enterprise"
)

type Limits struct {
	MaxDevices      int `json:"max_devices"`
	MaxSubscribers  int `json:"max_subscribers"`
	MaxUsers        int `json:"max_users"`
	MaxAPIRPM       int `json:"max_api_rpm"`
	MaxConnectors   int `json:"max_connectors"`
	AuditRetainDays int `json:"audit_retain_days"`
	TelemetryDays   int `json:"telemetry_days"`
}

type Usage struct {
	Devices     int `json:"devices"`
	Subscribers int `json:"subscribers"`
	Users       int `json:"users"`
	Connectors  int `json:"connectors"`
}

var plans = map[Plan]Limits{
	Community:    {MaxDevices: 10, MaxSubscribers: 500, MaxUsers: 3, MaxAPIRPM: 60, MaxConnectors: 3, AuditRetainDays: 30, TelemetryDays: 7},
	Professional: {MaxDevices: 200, MaxSubscribers: 10000, MaxUsers: 15, MaxAPIRPM: 600, MaxConnectors: 15, AuditRetainDays: 180, TelemetryDays: 30},
	Business:     {MaxDevices: 2000, MaxSubscribers: 100000, MaxUsers: 100, MaxAPIRPM: 3000, MaxConnectors: 60, AuditRetainDays: 365, TelemetryDays: 90},
	Enterprise:   {MaxDevices: -1, MaxSubscribers: -1, MaxUsers: -1, MaxAPIRPM: -1, MaxConnectors: -1, AuditRetainDays: 2555, TelemetryDays: 365},
}

func ForPlan(p Plan) Limits {
	if l, ok := plans[p]; ok {
		return l
	}
	return plans[Community]
}

// Violation names one exceeded limit (-1 = unlimited).
type Violation struct {
	Resource string `json:"resource"`
	Used     int    `json:"used"`
	Limit    int    `json:"limit"`
}

func Check(p Plan, u Usage) []Violation {
	l := ForPlan(p)
	pairs := []struct {
		name        string
		used, limit int
	}{
		{"devices", u.Devices, l.MaxDevices},
		{"subscribers", u.Subscribers, l.MaxSubscribers},
		{"users", u.Users, l.MaxUsers},
		{"connectors", u.Connectors, l.MaxConnectors},
	}
	var out []Violation
	for _, pr := range pairs {
		if pr.limit >= 0 && pr.used > pr.limit {
			out = append(out, Violation{Resource: pr.name, Used: pr.used, Limit: pr.limit})
		}
	}
	return out
}
