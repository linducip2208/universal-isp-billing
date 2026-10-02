package jobs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/universal-isp/platform/internal/jobs"
)

func TestIdempotencyAndDLQ(t *testing.T) {
	q := jobs.New()
	q.Register("fail", func(_ context.Context, _ *jobs.Job) error { return errors.New("x") })
	if err := q.Enqueue(&jobs.Job{ID: "1", Kind: "fail", IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(&jobs.Job{ID: "2", Kind: "fail", IdempotencyKey: "k1"}); err == nil {
		t.Fatal("duplicate should fail")
	}
	q.Drain(context.Background())
	if len(q.Dead()) != 1 {
		t.Fatalf("dead=%d", len(q.Dead()))
	}
}
