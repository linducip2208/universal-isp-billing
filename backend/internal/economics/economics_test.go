package economics_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/economics"
)

func TestSummarize(t *testing.T) {
	mrr := []economics.MoneyRow{
		{CustomerID: "c1", Amount: 15000000, POP: "bogor", Package: "50M"},
		{CustomerID: "c2", Amount: 15000000, POP: "bogor", Package: "50M"},
	}
	subs := []economics.SubRow{
		{ID: "s1", Status: "active"}, {ID: "s2", Status: "active"},
		{ID: "s3", Status: "suspended"}, {ID: "s4", Status: "terminated", Churned: true},
	}
	s := economics.Summarize(mrr, subs)
	if s.MRRcents != 30000000 || s.ARPUcents != 15000000 {
		t.Fatalf("s=%+v", s)
	}
	if s.RevenueByPOP["bogor"] != 30000000 {
		t.Fatalf("pop=%v", s.RevenueByPOP)
	}
}
