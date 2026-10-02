package lab_test

import (
	"context"
	"testing"

	"github.com/universal-isp/platform/internal/connectors/generic"
	"github.com/universal-isp/platform/internal/connectors/lab"
)

func TestLabSNMP(t *testing.T) {
	c := generic.NewSNMP("127.0.0.1:161", "public")
	res, err := lab.TestAndDiscover(context.Background(), c)
	if err != nil {
		t.Skipf("udp dial unavailable: %v", err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
}
