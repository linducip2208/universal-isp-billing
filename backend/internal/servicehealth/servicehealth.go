// Package servicehealth composes the premium per-subscriber health view:
// customer -> service -> RADIUS session -> NAS -> device -> OLT/ONU/optical
// -> incidents. Read-only composition over injected fetchers (store-backed
// in production); every section records its source for evidence.
package servicehealth

import "context"

type Section struct {
	Source  string         `json:"source"`
	Status  string         `json:"status"` // ok | degraded | down | unknown
	Detail  string         `json:"detail"`
	Metrics map[string]any `json:"metrics,omitempty"`
}

type Report struct {
	SubscriberID string    `json:"subscriber_id"`
	Overall      string    `json:"overall"`
	Sections     []Section `json:"sections"`
}

// Fetcher resolves one section. Implementations query store/connectors read-only.
type Fetcher func(ctx context.Context, subscriberID string) Section

// Compose runs fetchers in order and rolls up the worst status.
func Compose(ctx context.Context, subscriberID string, fetchers []Fetcher) Report {
	r := Report{SubscriberID: subscriberID, Overall: "ok"}
	rank := map[string]int{"ok": 0, "unknown": 1, "degraded": 2, "down": 3}
	for _, f := range fetchers {
		s := f(ctx, subscriberID)
		r.Sections = append(r.Sections, s)
		if rank[s.Status] > rank[r.Overall] {
			r.Overall = s.Status
		}
	}
	return r
}
