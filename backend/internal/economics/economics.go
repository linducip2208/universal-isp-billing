// Package economics: network business intelligence — MRR, ARPU, churn,
// revenue per POP/OLT/package, suspension rate. Pure aggregation over
// injected rows (store-backed in production); money stays integer cents.
package economics

type MoneyRow struct {
	CustomerID string `json:"customer_id"`
	Amount     int64  `json:"amount_cents"`
	POP        string `json:"pop,omitempty"`
	Package    string `json:"package,omitempty"`
}

type SubRow struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // active|suspended|terminated
	POP       string `json:"pop,omitempty"`
	Package   string `json:"package,omitempty"`
	NewPeriod bool   `json:"new_period"`
	Churned   bool   `json:"churned"`
}

type Summary struct {
	MRRcents      int64            `json:"mrr_cents"`
	ARPUcents     int64            `json:"arpu_cents"`
	ChurnPct      float64          `json:"churn_pct"`
	SuspensionPct float64          `json:"suspension_pct"`
	RevenueByPOP  map[string]int64 `json:"revenue_by_pop"`
	RevenueByPack map[string]int64 `json:"revenue_by_package"`
	ActiveSubs    int              `json:"active_subs"`
}

func Summarize(mrr []MoneyRow, subs []SubRow) Summary {
	s := Summary{RevenueByPOP: map[string]int64{}, RevenueByPack: map[string]int64{}}
	for _, r := range mrr {
		s.MRRcents += r.Amount
		s.RevenueByPOP[r.POP] += r.Amount
		s.RevenueByPack[r.Package] += r.Amount
	}
	var active, susp, churned, base int
	for _, sub := range subs {
		switch sub.Status {
		case "active":
			active++
		case "suspended":
			susp++
		}
		if sub.Churned {
			churned++
		}
		if !sub.NewPeriod {
			base++
		}
	}
	s.ActiveSubs = active
	if active > 0 {
		s.ARPUcents = s.MRRcents / int64(active)
	}
	if base > 0 {
		s.ChurnPct = float64(churned) / float64(base) * 100
	}
	if active+susp > 0 {
		s.SuspensionPct = float64(susp) / float64(active+susp) * 100
	}
	return s
}
