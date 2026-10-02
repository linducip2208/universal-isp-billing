package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/universal-isp/platform/internal/jobs"
)

// Runner hosts background workers: billing, provisioning, polling, alerts,
// webhooks, notifications, cleanup. Each worker drains its queue on a tick.
func Run(ctx context.Context, log *slog.Logger, q *jobs.Queue, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			q.Drain(ctx)
			log.Debug("worker tick")
		}
	}
}
