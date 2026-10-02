# Security review (2026-10-02, code audit)

Covered: PBKDF2-SHA256 (210k) + timing-safe compare; 90-day rotation flag;
jti revocation + logout; login throttle (5/min) + global/org rate limits;
RBAC + org scoping on every store query (live tenant-isolation test);
CORS allowlist; SSRF guard on generic HTTP; HMAC webhooks; API-key hashing;
AES-GCM vault; audit records; secrets absent from logs/responses (verified
by grep: no password/secret fields in API structs); SQL whitelists only
(no string-interpolated queries); React auto-escaping (XSS); command
injection: connectors use parameterized API words, no shell.
Gaps (tracked): session revocation is in-process (Redis backing PLANNED
for multi-instance); password rotation is flagged, not forced; MFA/WebAuthn
+ OIDC/SAML are readiness seams only; LTE: brute-force is per-IP throttle
(no distributed counter yet — needs Redis rollout).
