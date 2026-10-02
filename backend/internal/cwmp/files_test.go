package cwmp_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/cwmp"
)

func TestFileStaging(t *testing.T) {
	s := cwmp.NewServer()
	s.Users = map[string]string{"ops": "pw"}
	fs := cwmp.NewFileStore()
	h := s.FileHandler(fs)
	// unauthenticated upload -> 401
	up := httptest.NewRequest("PUT", "/files/fw.bin", strings.NewReader("bytes"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, up)
	if rec.Code != 401 {
		t.Fatalf("upload without auth: %d", rec.Code)
	}
	// authenticated upload
	up2 := httptest.NewRequest("PUT", "/files/fw.bin", strings.NewReader("firmware-bytes"))
	up2.SetBasicAuth("ops", "pw")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, up2)
	if rec2.Code != 201 {
		t.Fatalf("upload: %d %s", rec2.Code, rec2.Body.String())
	}
	// CPE download (no auth needed on GET — URL is capability-scoped)
	dl := httptest.NewRequest("GET", "/files/fw.bin", nil)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, dl)
	if rec3.Code != 200 || rec3.Body.String() != "firmware-bytes" {
		t.Fatalf("download: %d %q", rec3.Code, rec3.Body.String())
	}
	miss := httptest.NewRequest("GET", "/files/nope", nil)
	rec4 := httptest.NewRecorder()
	h.ServeHTTP(rec4, miss)
	if rec4.Code != 404 {
		t.Fatalf("missing: %d", rec4.Code)
	}
}

func TestStormGuard(t *testing.T) {
	s := cwmp.NewServer()
	shed := 0
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest("POST", "/acs", strings.NewReader(informXML))
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code == 204 {
			shed++
		}
	}
	if shed == 0 {
		t.Fatal("storming CPE must be shed")
	}
	if suspects := s.StormSuspects(); len(suspects) != 1 || suspects[0] != "SN-001" {
		t.Fatalf("suspects=%v", suspects)
	}
}
