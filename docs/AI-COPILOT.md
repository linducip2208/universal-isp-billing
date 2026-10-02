# AI Copilot

Read-only by construction (`internal/copilot`): role-gated Ask, tool
allowlist (`radius_lookup`, `incident_summary`, ...), evidence attached to
every claim, execution-like intents refused, `executed_anything` always
false in responses. Recommendations are proposals requiring normal change
management (approval + audit). No shell, no device writes, no arbitrary code.
UI: /copilot page. LLM plugs behind the tool-selection step; default is a
deterministic analyzer (testable without a model).
