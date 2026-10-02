package customers_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/customers"
)

func TestValidateAndTransition(t *testing.T) {
	c := &customers.Customer{Name: "Budi", Email: "bad"}
	if err := customers.Validate(c); err == nil {
		t.Fatal("bad email should fail")
	}
	c.Email = "budi@example.id"
	if err := customers.Validate(c); err != nil {
		t.Fatal(err)
	}
	if err := customers.Transition(c, customers.StatusClosed); err != nil {
		// zero status is "" not active; set active first
		c.Status = customers.StatusActive
		if err := customers.Transition(c, customers.StatusSuspended); err != nil {
			t.Fatal(err)
		}
		if err := customers.Transition(c, customers.StatusActive); err != nil {
			t.Fatal(err)
		}
	}
}
