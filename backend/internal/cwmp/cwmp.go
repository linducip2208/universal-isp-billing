// Package cwmp implements a functional TR-069 subset: CPE Inform ingestion,
// parameter get/set, and reboot tasks over SOAP/HTTP. It implements the
// tr069.ACS seam. Full session features (retry storms, digest/client-cert
// auth, file transfer, scheduling) are PLANNED — this is PARTIAL and says so.
package cwmp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/universal-isp/platform/internal/tr069"
)

// --- SOAP envelope parsing (minimal, method-dispatch only) ---

type envelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Inform                *informReq `xml:"Inform"`
		TransferComplete      *struct{}  `xml:"TransferComplete"`
		GetRPCMethodsResponse *struct{}  `xml:"GetRPCMethodsResponse"`
	} `xml:"Body"`
}

type informReq struct {
	DeviceID struct {
		Manufacturer string `xml:"Manufacturer"`
		OUI          string `xml:"OUI"`
		ProductClass string `xml:"ProductClass"`
		SerialNumber string `xml:"SerialNumber"`
	} `xml:"DeviceId"`
	Events []struct {
		EventCode string `xml:"EventCode"`
	} `xml:"Event>EventStruct"`
	Params []struct {
		Name  string `xml:"Name"`
		Value string `xml:"Value"`
	} `xml:"ParameterList>ParameterValueStruct"`
}

// Server is an ACS: device registry + task queue + HTTP endpoint.
type Server struct {
	mu      sync.RWMutex
	devices map[string]tr069.DeviceID // serial -> id
	last    map[string]time.Time
	tasks   map[string][]tr069.Task
	params  map[string]map[string]string // serial -> name -> value (last known)
}

func NewServer() *Server {
	return &Server{devices: map[string]tr069.DeviceID{}, last: map[string]time.Time{},
		tasks: map[string][]tr069.Task{}, params: map[string]map[string]string{}}
}

func (s *Server) IngestInform(_ context.Context, in tr069.Inform) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[in.Device.SerialNumber] = in.Device
	s.last[in.Device.SerialNumber] = in.Received
	if s.params[in.Device.SerialNumber] == nil {
		s.params[in.Device.SerialNumber] = map[string]string{}
	}
	for k, v := range in.Params {
		s.params[in.Device.SerialNumber][k] = v
	}
	return nil
}

func (s *Server) EnqueueTask(_ context.Context, t tr069.Task) error {
	if t.Serial == "" || t.Action == "" {
		return fmt.Errorf("serial and action required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.Serial] = append(s.tasks[t.Serial], t)
	return nil
}

func (s *Server) PendingTasks(_ context.Context, serial string) ([]tr069.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]tr069.Task{}, s.tasks[serial]...), nil
}

func (s *Server) AckTask(_ context.Context, serial, action string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rest := s.tasks[serial][:0]
	for _, t := range s.tasks[serial] {
		if t.Action != action {
			rest = append(rest, t)
		}
	}
	s.tasks[serial] = rest
	return nil
}

func (s *Server) deviceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.devices)
}

// ServeHTTP handles CPE POSTs: Inform -> InformResponse (+ pending tasks as
// GetParameterValues/SetParameterValues/Reboot); others -> empty response.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "CWMP requires POST", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		// Empty POST = CPE ready for more requests; answer session end.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var env envelope
	if err := xml.Unmarshal(body, &env); err != nil || env.Body.Inform == nil {
		// Unknown/unsupported RPC: honest empty (session continues).
		w.WriteHeader(http.StatusNoContent)
		return
	}
	inf := env.Body.Inform
	in := tr069.Inform{
		Device: tr069.DeviceID{Manufacturer: inf.DeviceID.Manufacturer, OUI: inf.DeviceID.OUI,
			ProductClass: inf.DeviceID.ProductClass, SerialNumber: inf.DeviceID.SerialNumber},
		Received: time.Now().UTC(),
		Params:   map[string]string{},
	}
	for _, e := range inf.Events {
		in.Event += e.EventCode + " "
	}
	in.Event = strings.TrimSpace(in.Event)
	if in.Event == "" {
		in.Event = "PERIODIC"
	}
	for _, p := range inf.Params {
		in.Params[p.Name] = p.Value
	}
	_ = s.IngestInform(r.Context(), in)
	serial := in.Device.SerialNumber
	tasks, _ := s.PendingTasks(r.Context(), serial)
	w.Header().Set("Content-Type", "text/xml")
	if len(tasks) == 0 {
		_, _ = w.Write([]byte(soapInformResponse()))
		return
	}
	t := tasks[0]
	switch t.Action {
	case "GetParameterValues":
		names := []string{"Device.DeviceInfo.SoftwareVersion"}
		if v, ok := t.Params["names"]; ok && v != "" {
			names = strings.Split(v, ",")
		}
		_, _ = w.Write([]byte(soapGetParams(names)))
	case "SetParameterValues":
		var names []string
		var vals []string
		for k, v := range t.Params {
			names = append(names, k)
			vals = append(vals, v)
		}
		_, _ = w.Write([]byte(soapSetParams(names, vals)))
	case "Reboot":
		_, _ = w.Write([]byte(soapReboot()))
		_ = s.AckTask(r.Context(), serial, "Reboot") // fire-and-forget RPC
	default:
		_, _ = w.Write([]byte(soapInformResponse()))
	}
}

func soapInformResponse() string {
	return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><cwmp:InformResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><MaxEnvelopes>1</MaxEnvelopes></cwmp:InformResponse></soap:Body></soap:Envelope>`
}

func soapGetParams(names []string) string {
	var sb strings.Builder
	sb.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><cwmp:GetParameterValues xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><ParameterNames>`)
	for _, n := range names {
		sb.WriteString(`<string>` + xmlEscape(n) + `</string>`)
	}
	sb.WriteString(`</ParameterNames></cwmp:GetParameterValues></soap:Body></soap:Envelope>`)
	return sb.String()
}

func soapSetParams(names, vals []string) string {
	var sb strings.Builder
	sb.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><cwmp:SetParameterValues xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><ParameterList>`)
	for i, n := range names {
		v := ""
		if i < len(vals) {
			v = vals[i]
		}
		sb.WriteString(`<ParameterValueStruct><Name>` + xmlEscape(n) + `</Name><Value>` + xmlEscape(v) + `</Value></ParameterValueStruct>`)
	}
	sb.WriteString(`</ParameterList><ParameterKey></ParameterKey></cwmp:SetParameterValues></soap:Body></soap:Envelope>`)
	return sb.String()
}

func soapReboot() string {
	return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><cwmp:Reboot xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><CommandKey></CommandKey></cwmp:Reboot></soap:Body></soap:Envelope>`
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
