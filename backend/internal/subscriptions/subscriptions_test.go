package subscriptions_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/subscriptions"
)

func TestLifecycle(t *testing.T) {
	s := &subscriptions.Subscription{Status: subscriptions.Pending}
	if err := s.Suspend(); err == nil {
		t.Fatal("pending must not suspend")
	}
	if err := s.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := s.Suspend(); err != nil {
		t.Fatal(err)
	}
	if err := s.Activate(); err != nil {
		t.Fatal(err)
	}
	if err := s.Terminate(); err != nil {
		t.Fatal(err)
	}
	if err := s.Activate(); err == nil {
		t.Fatal("terminated must not activate")
	}
}
