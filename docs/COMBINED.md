# Monitoring / Security / Deployment / Development / API

- Monitoring: per-device polling intervals, CPU/mem/uptime/iface/loss/latency,
  AP/OLT/ONU/session telemetry into `traffic_samples`/`interface_samples`
  (partition-ready), alerts via automation engine.
- Security: RBAC, JWT+API keys, rate limit, login throttle, validation,
  audit logs, AES-GCM secrets, SSRF guards, command-injection-safe connectors.
- Deployment: Linux systemd + aaPanel-friendly; Docker optional. See deployments/.
- Development: `go test ./...`, `go vet ./...`, frontend `npm run build`.
- API: `/api/v1/*`, OpenAPI at `backend/openapi/openapi.yaml`.
