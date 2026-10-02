package jobs_test

import (
	"context"
	"testing"

	"github.com/universal-isp/platform/internal/jobs"
)

// BenchmarkDrain measures in-process queue throughput (1000 no-op jobs).
func BenchmarkDrain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := jobs.New()
		q.Register("noop", func(_ context.Context, _ *jobs.Job) error { return nil })
		for j := 0; j < 1000; j++ {
			_ = q.Enqueue(&jobs.Job{ID: string(rune(j)), Kind: "noop"})
		}
		b.StartTimer()
		q.Drain(context.Background())
		b.StopTimer()
		if q.Pending() != 0 {
			b.Fatal("drain incomplete")
		}
	}
}
