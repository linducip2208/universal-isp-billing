package cwmp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/cwmp"
	"github.com/universal-isp/platform/internal/tr069"
)

const informXML = `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><cwmp:Inform xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><DeviceId><Manufacturer>Acme</Manufacturer><OUI>001122</OUI><ProductClass>ONT</ProductClass><SerialNumber>SN-001</SerialNumber></DeviceId><Event><EventStruct><EventCode>2 PERIODIC</EventCode></EventStruct></Event><ParameterList><ParameterValueStruct><Name>Device.DeviceInfo.SoftwareVersion</Name><Value>1.2.3</Value></ParameterValueStruct></ParameterList></cwmp:Inform></soap:Body></soap:Envelope>`

func TestInformAndRebootTask(t *testing.T) {
	s := cwmp.NewServer()
	// 1. CPE Inform, no tasks -> InformResponse
	req := httptest.NewRequest("POST", "/acs", strings.NewReader(informXML))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "InformResponse") {
		t.Fatalf("inform: %d %s", rec.Code, rec.Body.String())
	}
	// 2. enqueue reboot, next Inform triggers Reboot RPC
	if err := s.EnqueueTask(context.Background(), tr069.Task{Serial: "SN-001", Action: "Reboot"}); err != nil {
		t.Fatal(err)
	}
	req2 := httptest.NewRequest("POST", "/acs", strings.NewReader(informXML))
	rec2 := httptest.NewRecorder()
	s.ServeHTTP(rec2, req2)
	if !strings.Contains(rec2.Body.String(), "Reboot") {
		t.Fatalf("reboot rpc missing: %s", rec2.Body.String())
	}
	// 3. reboot acked automatically; GPV task served next
	if err := s.EnqueueTask(context.Background(), tr069.Task{Serial: "SN-001", Action: "GetParameterValues", Params: map[string]string{"names": "Device.DeviceInfo.UpTime"}}); err != nil {
		t.Fatal(err)
	}
	req3 := httptest.NewRequest("POST", "/acs", strings.NewReader(informXML))
	rec3 := httptest.NewRecorder()
	s.ServeHTTP(rec3, req3)
	if !strings.Contains(rec3.Body.String(), "GetParameterValues") || !strings.Contains(rec3.Body.String(), "UpTime") {
		t.Fatalf("gpv missing: %s", rec3.Body.String())
	}
	// 4. empty POST + GET honesty
	reqE := httptest.NewRequest("POST", "/acs", nil)
	recE := httptest.NewRecorder()
	s.ServeHTTP(recE, reqE)
	if recE.Code != 204 {
		t.Fatalf("empty post: %d", recE.Code)
	}
	reqG := httptest.NewRequest("GET", "/acs", nil)
	recG := httptest.NewRecorder()
	s.ServeHTTP(recG, reqG)
	if recG.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get: %d", recG.Code)
	}
}
