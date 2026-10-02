// Package soc: ISP security-operations detectors over audit/login event
// feeds. Deterministic rules (no ML black box): brute-force bursts, config
// change bursts, off-hours privileged access, API abuse. Findings cite the
// source events as evidence and can feed incidents/alerts.
package soc

import (
	"time"
)

type Event struct {
	At     time.Time `json:"at"`
	Actor  string    `json:"actor"`
	Action string    `json:"action"` // login.failed | login.ok | config.change | api.4xx | device.login
	IP     string    `json:"ip,omitempty"`
}

type Finding struct {
	Rule     string    `json:"rule"`
	Severity string    `json:"severity"`
	Actor    string    `json:"actor"`
	Count    int       `json:"count"`
	Window   string    `json:"window"`
	At       time.Time `json:"at"`
}

// burst finds actors with >= threshold matching actions inside window.
func burst(events []Event, actions map[string]bool, threshold int, window time.Duration, rule, sev string) []Finding {
	byActor := map[string][]time.Time{}
	for _, e := range events {
		if actions[e.Action] {
			byActor[e.Actor] = append(byActor[e.Actor], e.At)
		}
	}
	var out []Finding
	for actor, ts := range byActor {
		for i := range ts {
			n := 0
			for j := i; j < len(ts) && ts[j].Sub(ts[i]) < window; j++ {
				n++
			}
			if n >= threshold {
				out = append(out, Finding{Rule: rule, Severity: sev, Actor: actor, Count: n, Window: window.String(), At: ts[i]})
				break
			}
		}
	}
	return out
}

// Analyze runs all detectors over the feed.
func Analyze(events []Event) []Finding {
	var out []Finding
	out = append(out, burst(events, map[string]bool{"login.failed": true}, 5, 5*time.Minute, "brute-force", "critical")...)
	out = append(out, burst(events, map[string]bool{"config.change": true}, 10, 10*time.Minute, "config-churn", "major")...)
	out = append(out, burst(events, map[string]bool{"api.4xx": true}, 50, 5*time.Minute, "api-abuse", "major")...)
	var off []Finding
	for _, e := range events {
		if e.Action == "device.login" && (e.At.Hour() < 6 || e.At.Hour() >= 23) {
			off = append(off, Finding{Rule: "off-hours-device-login", Severity: "minor", Actor: e.Actor, Count: 1, At: e.At})
		}
	}
	return append(out, off...)
}
