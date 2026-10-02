package ipam_test

import (
	"net/netip"
	"testing"

	"github.com/universal-isp/platform/internal/ipam"
)

func BenchmarkAllocateNext(b *testing.B) {
	p := ipam.NewPool("p", "o", "t", netip.MustParsePrefix("10.0.0.0/8"), 0, "")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.AllocateNext("owner"); err != nil {
			b.Fatal(err)
		}
	}
}
