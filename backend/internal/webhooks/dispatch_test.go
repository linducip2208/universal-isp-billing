package webhooks_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/webhooks"
)

func TestDispatchAndReplay(t *testing.T) {
	var gotBody []byte
	var gotSig string
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody, gotSig = buf, r.Header.Get("X-ISP-Signature")
		w.WriteHeader(200)
	}))
	defer ok.Close()
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer fail.Close()

	d := webhooks.NewDispatcher([]webhooks.Endpoint{
		{URL: ok.URL, Secret: "s", Events: []string{"invoice.*"}},
		{URL: fail.URL, Secret: "s", Events: []string{"invoice.overdue"}},
		{URL: ok.URL, Secret: "s", Events: []string{"device.down"}},
	})
	res := d.Dispatch(context.Background(), "invoice.overdue", []byte(`{"id":1}`))
	if len(res) != 2 {
		t.Fatalf("want 2 deliveries, got %d", len(res))
	}
	if string(gotBody) != `{"id":1}` || gotSig == "" {
		t.Fatalf("body=%q sig=%q", gotBody, gotSig)
	}
	if len(d.History()) != 2 {
		t.Fatalf("history=%d", len(d.History()))
	}
	dl, found := d.Replay(context.Background(), ok.URL, "invoice.overdue", []byte(`{"id":1}`))
	if !found || dl.Status != 200 {
		t.Fatalf("replay=%+v found=%v", dl, found)
	}
	if _, found := d.Replay(context.Background(), "http://nope", "x", nil); found {
		t.Fatal("unknown endpoint must not replay")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = ctx
}
