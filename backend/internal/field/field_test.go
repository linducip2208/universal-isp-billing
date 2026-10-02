package field_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/field"
)

func TestTicketMachine(t *testing.T) {
	tk := &field.Ticket{Status: field.TicketOpen}
	if err := field.Advance(tk, field.TicketInProgress); err == nil {
		t.Fatal("open->in_progress must fail")
	}
	for _, s := range []field.TicketStatus{field.TicketAssigned, field.TicketInProgress, field.TicketResolved, field.TicketClosed} {
		if err := field.Advance(tk, s); err != nil {
			t.Fatalf("->%s: %v", s, err)
		}
	}
	if err := field.Advance(tk, field.TicketOpen); err == nil {
		t.Fatal("closed is terminal")
	}
}
