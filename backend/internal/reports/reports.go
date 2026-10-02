// Package reports: revenue, subscriber, and network summary aggregation
// over injectable row providers (DB-backed in production).
package reports

type RevenuePoint struct {
	Period      string `json:"period"`
	Invoiced    int64  `json:"invoiced_cents"`
	Collected   int64  `json:"collected_cents"`
	Outstanding int64  `json:"outstanding_cents"`
}

type SubscriberPoint struct {
	Period    string `json:"period"`
	Active    int    `json:"active"`
	Suspended int    `json:"suspended"`
	New       int    `json:"new"`
	Churned   int    `json:"churned"`
}

func Outstanding(inv, paid int64) int64 {
	if paid >= inv {
		return 0
	}
	return inv - paid
}

func CollectionRate(invoiced, collected int64) float64 {
	if invoiced <= 0 {
		return 0
	}
	return float64(collected) / float64(invoiced) * 100
}
