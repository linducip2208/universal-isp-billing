package provisioning

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

// Step is one idempotent workflow step.
type Step struct {
	Name string
	Run  func(ctx context.Context, w *Workflow) error
}

type Workflow struct {
	ID             string
	SubscriptionID string
	DeviceID       string
	Connector      sdk.NetworkConnector
	Provision      sdk.ProvisionRequest
	Attempts       map[string]int
	Events         []string
}

func (w *Workflow) log(format string, args ...any) {
	w.Events = append(w.Events, fmt.Sprintf(format, args...))
}

// Execute runs steps with retry + exponential backoff + timeout per step.
func Execute(ctx context.Context, w *Workflow, steps []Step) error {
	if w.Attempts == nil {
		w.Attempts = map[string]int{}
	}
	for _, s := range steps {
		if err := runStep(ctx, w, s); err != nil {
			return fmt.Errorf("step %s: %w", s.Name, err)
		}
	}
	return nil
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
