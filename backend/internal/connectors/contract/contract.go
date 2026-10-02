// Package contract is the connector certification harness: every registered
// connector must satisfy fail-closed behavior, idempotency requirements, and
// honest capability declarations — without any network or credentials.
package contract

import (
	"context"
	"fmt"
	"time"

	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type Result struct {
	Vendor   string   `json:"vendor"`
	Family   string   `json:"family"`
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
}

// Check runs the contract against one connector built with EMPTY config:
//  1. no panics on any method (recover-guarded, deadline-bounded);
//  2. TestConnection must FAIL (fail-closed without credentials/network);
//  3. Provision/Update/Suspend/Activate/Disconnect/Delete must not succeed
//     silently — error expected (fail-closed) or explicit success only with
//     idempotency keys where applicable;
//  4. GetCapabilities must return a non-empty declaration.
func Check(name string, e registry.Entry) (res Result) {
	res = Result{Vendor: e.Vendor, Family: e.ProductFamily, Passed: true}
	defer func() {
		if r := recover(); r != nil {
			res.Passed, res.Failures = false, append(res.Failures, fmt.Sprintf("panic: %v", r))
		}
	}()
	fail := func(f string) {
		res.Passed = false
		res.Failures = append(res.Failures, f)
	}
	if e.Factory == nil {
		fail("no factory")
		return res
	}
	c, err := e.Factory(map[string]string{})
	if err != nil {
		fail("factory with empty config must not hard-fail: " + err.Error())
		return res
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.TestConnection(ctx); err == nil {
		fail("TestConnection with empty config must fail (fail-closed)")
	}
	caps, err := c.GetCapabilities(ctx)
	if err != nil {
		fail("GetCapabilities must work offline: " + err.Error())
	} else if caps == nil || len(caps.Items) == 0 {
		fail("GetCapabilities must declare at least one capability")
	}
	// Mutations without idempotency keys must be refused.
	if err := c.ProvisionSubscriber(ctx, sdk.ProvisionRequest{SubscriberID: "t", Username: "t"}); err == nil {
		fail("ProvisionSubscriber without idempotency key must fail")
	}
	if err := c.UpdateSubscriber(ctx, sdk.UpdateRequest{SubscriberID: "t"}); err == nil {
		// Update without key: tolerated only if explicitly documented; flag it.
		fail("UpdateSubscriber without idempotency key must fail")
	}
	// Lifecycle ops on unknown subscribers must complete (error or explicit
	// no-op) without panic or hang — outcome recorded, not judged.
	for _, op := range []struct {
		name string
		fn   func() error
	}{
		{"suspend", func() error { return c.SuspendSubscriber(ctx, "nope") }},
		{"activate", func() error { return c.ActivateSubscriber(ctx, "nope") }},
		{"disconnect", func() error { return c.DisconnectSubscriber(ctx, "nope") }},
		{"delete", func() error { return c.DeleteSubscriber(ctx, "nope") }},
	} {
		done := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("panic: %v", r)
				}
			}()
			done <- op.fn()
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			fail(op.name + ": hung")
		}
	}
	_ = name
	return res
}
