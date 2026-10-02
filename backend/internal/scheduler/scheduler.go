// Package scheduler runs named interval tasks (billing generation,
// reconciliation, polling fan-out, cleanup). Tasks must be idempotent;
// overlapping runs of the same task are skipped, never parallelized.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Task struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

type Scheduler struct {
	log   *slog.Logger
	tasks []Task
}

func New(log *slog.Logger, tasks []Task) *Scheduler {
	return &Scheduler{log: log, tasks: tasks}
}

func (s *Scheduler) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, t := range s.tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			tick := time.NewTicker(t.Interval)
			defer tick.Stop()
			running := false
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					if running {
						s.log.Warn("scheduler skip overlap", slog.String("task", t.Name))
						continue
					}
					running = true
					if err := t.Run(ctx); err != nil {
						s.log.Error("scheduler task failed", slog.String("task", t.Name), slog.String("err", err.Error()))
					}
					running = false
				}
			}
		}(t)
	}
	wg.Wait()
}
