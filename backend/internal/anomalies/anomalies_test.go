package anomalies_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/anomalies"
)

func TestZScore(t *testing.T) {
	var s []anomalies.Point
	for i := 0; i < 20; i++ {
		s = append(s, anomalies.Point{At: int64(i), Value: 50})
	}
	s = append(s, anomalies.Point{At: 20, Value: 200})
	f := anomalies.ZScore{Sigma: 3, MinData: 10}.Check("cpu", s)
	if len(f) != 1 || f[0].At != 20 {
		t.Fatalf("findings=%+v", f)
	}
	if len(anomalies.ZScore{Sigma: 3, MinData: 30}.Check("cpu", s)) != 0 {
		t.Fatal("too little data must not flag")
	}
}

func TestThreshold(t *testing.T) {
	s := []anomalies.Point{{At: 1, Value: 10}, {At: 2, Value: 95}}
	f := anomalies.Threshold{Low: 0, High: 90}.Check("mem", s)
	if len(f) != 1 || f[0].At != 2 {
		t.Fatalf("findings=%+v", f)
	}
}
