package audit

import "time"

type Record struct {
	Actor      string         `json:"actor"`
	Org        string         `json:"organization"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	Before     map[string]any `json:"before,omitempty"`
	After      map[string]any `json:"after,omitempty"`
	IP         string         `json:"ip"`
	UserAgent  string         `json:"user_agent"`
	Result     string         `json:"result"`
	At         time.Time      `json:"at"`
}
