package radius

// Server.Meter wires packet counters into the shared Prometheus registry
// (nil-safe: unmetered when absent). See radius.go for hook points.
func (s *Server) meterInc(name string, labels map[string]string) {
	if s.Meter == nil {
		return
	}
	s.Meter.Inc(name, labels)
}

// statusName maps Acct-Status-Type octets to labels (unknown -> "unknown").
func statusName(status string) string {
	switch status {
	case string([]byte{1}):
		return "start"
	case string([]byte{2}):
		return "stop"
	case string([]byte{3}):
		return "interim"
	case string([]byte{7}):
		return "accounting-on"
	case string([]byte{8}):
		return "accounting-off"
	default:
		return "unknown"
	}
}
