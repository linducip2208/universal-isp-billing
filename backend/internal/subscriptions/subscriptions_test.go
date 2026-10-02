package subscriptions_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/subscriptions"
)

func TestLifecycle(t *testing.T) {
	s := &subscriptions.Subscription{Status: subscriptions.Prospect}
	if err := s.Suspend(); err == nil {
		t.Fatal("prospect must not suspend")
	}
	if err := s.Transition(subscriptions.Pending); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(subscriptions.Provisioning); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(subscriptions.Active); err != nil {
		t.Fatal(err)
	}
	if err := s.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(subscriptions.Grace); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(subscriptions.Reactivating); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(subscriptions.Active); err != nil {
		t.Fatal(err)
	}
	if err := s.Terminate(); err != nil {
		t.Fatal(err)
	}
	if err := s.Activate(); err == nil {
		t.Fatal("terminated must not activate")
	}
	// pre-service cancellation path
	s2 := &subscriptions.Subscription{Status: subscriptions.Pending}
	if err := s2.Terminate(); err == nil {
		t.Fatal("pending must use Cancelled, not Terminate")
	}
	if err := s2.Transition(subscriptions.Cancelled); err != nil {
		t.Fatal(err)
	}
}
