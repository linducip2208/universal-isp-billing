package provisioning

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

// Step is one idempotent workflow step with an optional compensation
// (rollback action executed in reverse order when a later step fails).
type Step struct {
	Name       string
	Run        func(ctx context.Context, w *Workflow) error
	Compensate func(ctx context.Context, w *Workflow) error
}

// RunRecord is the audit trail of one step execution.
type RunRecord struct {
	Step     string        `json:"step"`
	Attempts int           `json:"attempts"`
	Duration time.Duration `json:"duration_ms"`
	Result   string        `json:"result"` // ok | failed | compensated
	Error    string        `json:"error,omitempty"`
}

type Workflow struct {
	ID             string
	SubscriptionID string
	DeviceID       string
	Connector      sdk.NetworkConnector
	Provision      sdk.ProvisionRequest
	Attempts       map[string]int
	Events         []string
	RunLog         []RunRecord
}

func (w *Workflow) log(format string, args ...any) {
	w.Events = append(w.Events, fmt.Sprintf(format, args...))
}

// Execute runs steps with retry + exponential backoff + timeout per step.
// Every attempt is recorded in w.RunLog. On failure, compensations of
// completed steps run in reverse order (rollback), each also recorded.
func Execute(ctx context.Context, w *Workflow, steps []Step) error {
	if w.Attempts == nil {
		w.Attempts = map[string]int{}
	}
	var done []Step
	for _, s := range steps {
		start := time.Now()
		rec := RunRecord{Step: s.Name}
		if err := runStep(ctx, w, s); err != nil {
			rec.Attempts = w.Attempts[s.Name]
			rec.Duration = time.Since(start)
			rec.Result = "failed"
			rec.Error = err.Error()
			w.RunLog = append(w.RunLog, rec)
			w.compensate(ctx, done)
			return fmt.Errorf("step %s: %w", s.Name, err)
		}
		rec.Attempts = w.Attempts[s.Name]
		rec.Duration = time.Since(start)
		rec.Result = "ok"
		w.RunLog = append(w.RunLog, rec)
		done = append(done, s)
	}
	return nil
}

func (w *Workflow) compensate(ctx context.Context, done []Step) {
	for i := len(done) - 1; i >= 0; i-- {
		s := done[i]
		if s.Compensate == nil {
			continue
		}
		start := time.Now()
		rec := RunRecord{Step: s.Name + ":compensate", Attempts: 1}
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := s.Compensate(cctx, w)
		cancel()
		rec.Duration = time.Since(start)
		if err != nil {
			rec.Result = "failed"
			rec.Error = err.Error()
		} else {
			rec.Result = "compensated"
		}
		w.RunLog = append(w.RunLog, rec)
	}
}

func runStep(ctx context.Context, w *Workflow, s Step) error {
	const maxAttempts = 5
	backoff := 200 * time.Millisecond
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		w.Attempts[s.Name] = attempt
		stepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = s.Run(stepCtx, w)
		cancel()
		if err == nil {
			w.log("step %s ok (attempt %d)", s.Name, attempt)
			return nil
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff = time.Duration(math.Min(float64(backoff*2), float64(8*time.Second)))
	}
	return err
}

// ProvisionSubscriberWorkflow builds the canonical 10-step workflow.
func ProvisionSubscriberWorkflow() []Step {
	noop := func(name string) Step {
		return Step{Name: name, Run: func(ctx context.Context, w *Workflow) error { return nil }}
	}
	steps := []Step{
		noop("validate_subscription"),
		noop("validate_device"),
		noop("discover_capabilities"),
		noop("select_connector"),
		noop("generate_plan"),
		{Name: "execute", Run: func(ctx context.Context, w *Workflow) error {
			if w.Connector == nil {
				return fmt.Errorf("no connector selected")
			}
			return w.Connector.ProvisionSubscriber(ctx, w.Provision)
		}},
		noop("verify"),
		noop("record_result"),
		noop("audit"),
		noop("emit_event"),
	}
	return steps
}
