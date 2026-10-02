# Security

RBAC (6 default roles), HS256 JWT (OIDC-ready), API keys (hash-stored),
per-IP rate limiting, SSRF guards on generic HTTP, AES-256-GCM credential
vault with rotation seam, masked secrets in API, audit logs on network
commands, SQL parameterization, input validation. SSH/CLI connectors quote
arguments; never interpolate untrusted input.
