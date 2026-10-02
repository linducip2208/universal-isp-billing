// Package timezone: organization-aware business time. Timestamps are stored
// as UTC timestamptz; presentation converts to the org/user zone.
// Development default is Asia/Jakarta; never hardcode offsets in logic.
package timezone

import (
	"time"
)

const DefaultZone = "Asia/Jakarta"

// Load validates an IANA name, falling back to DefaultZone (never UTC
// silently for business display — the fallback is explicit and logged
// by callers via the returned bool).
func Load(name string) (*time.Location, bool) {
	if name == "" {
		name = DefaultZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc, _ = time.LoadLocation(DefaultZone)
		return loc, false
	}
	return loc, true
}

// Business formats t for operators in the org zone (ISO + zone abbrev).
func Business(t time.Time, zone string) string {
	loc, _ := Load(zone)
	return t.In(loc).Format("2006-01-02 15:04:05 MST")
}
