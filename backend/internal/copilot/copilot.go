// Package copilot: AI-assistant orchestration seam with hard guardrails.
// The copilot is READ-ONLY by construction: it selects from an allowlist of
// read tools, cites evidence (source references) for every network claim,
// and can only PROPOSE actions (never execute). An LLM plugs in behind the
// Responder interface; the default responder is a deterministic analyzer so
// behavior is testable without any model.
//
// Pipeline: question -> permission check -> tool selection -> read-only data
// -> analysis -> evidence -> recommendation -> (optional approval) -> audit.
package copilot

import (
	"context"
	"errors"
	"strings"
)

// Evidence backs one claim: where the data came from.
type Evidence struct {
	Source string `json:"source"` // e.g. "radius_sessions", "alerts", "topology"
	Ref    string `json:"ref"`
	Detail string `json:"detail"`
}

type Recommendation struct {
	Action     string `json:"action"` // proposed only, e.g. "suspend s-1"
	Requires   string `json:"requires_approval"`
	VerifyStep string `json:"verify"`
}

type Answer struct {
	Summary          string           `json:"summary"`
	Findings         []string         `json:"findings"`
	Evidence         []Evidence       `json:"evidence"`
	Recommendations  []Recommendation `json:"recommendations"`
	ExecutedAnything bool             `json:"executed_anything"` // always false
}

// Tool is a read-only data source. Name allowlist enforced by Orchestrator.
type Tool interface {
	Name() string
	Query(ctx context.Context, args map[string]string) ([]Evidence, error)
}

// Orchestrator routes questions to tools; unknown/exec intents are refused.
type Orchestrator struct {
	tools map[string]Tool
}

func New(tools ...Tool) *Orchestrator {
	o := &Orchestrator{tools: map[string]Tool{}}
	for _, t := range tools {
		o.tools[t.Name()] = t
	}
	return o
}

var execWords = []string{"run ", "execute", "delete ", "reboot", "configure", "ssh ", "shell", "drop table"}

// Ask answers read-only questions. Anything resembling execution is refused.
func (o *Orchestrator) Ask(ctx context.Context, roles []string, question string) (*Answer, error) {
	if !canAsk(roles) {
		return nil, errors.New("copilot: role not permitted")
	}
	low := strings.ToLower(question)
	for _, w := range execWords {
		if strings.Contains(low, w) {
			return &Answer{Summary: "refused: copilot is read-only; file a change request instead",
				ExecutedAnything: false}, nil
		}
	}
	ans := &Answer{ExecutedAnything: false}
	// Keyword tool selection (an LLM would do this semantically).
	if strings.Contains(low, "session") || strings.Contains(low, "subscriber") || strings.Contains(low, "radius") {
		if t, ok := o.tools["radius_lookup"]; ok {
			ev, err := t.Query(ctx, map[string]string{"q": question})
			if err == nil {
				ans.Evidence = append(ans.Evidence, ev...)
			}
		}
	}
	if strings.Contains(low, "incident") || strings.Contains(low, "alert") || strings.Contains(low, "down") || strings.Contains(low, "outage") {
		if t, ok := o.tools["incident_summary"]; ok {
			ev, err := t.Query(ctx, map[string]string{"q": question})
			if err == nil {
				ans.Evidence = append(ans.Evidence, ev...)
			}
		}
	}
	if len(ans.Evidence) == 0 {
		ans.Summary = "no evidence found for this question in connected read tools"
		return ans, nil
	}
	ans.Summary = "analysis from platform read tools (evidence cited below)"
	for _, e := range ans.Evidence {
		ans.Findings = append(ans.Findings, e.Source+": "+e.Detail)
	}
	return ans, nil
}

func canAsk(roles []string) bool {
	for _, r := range roles {
		switch r {
		case "superadmin", "admin", "noc", "support":
			return true
		}
	}
	return false
}
