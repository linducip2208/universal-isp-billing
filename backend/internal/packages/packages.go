// Package packages defines internet packages and bandwidth profiles.
// Prices are int64 minor units (cents) — never float.
package packages

import "errors"

type Profile struct {
	ID       string `json:"id"`
	OrgID    string `json:"org_id"`
	Name     string `json:"name"`
	DownMbps int    `json:"down_mbps"`
	UpMbps   int    `json:"up_mbps"`
}

type Package struct {
	ID         string `json:"id"`
	OrgID      string `json:"org_id"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	ProfileID  string `json:"profile_id"`
	Service    string `json:"service"` // pppoe | ipoe | hotspot | ftth
}

func Validate(p *Package) error {
	if p.Name == "" {
		return errors.New("name required")
	}
	if p.PriceCents < 0 {
		return errors.New("price must be >= 0")
	}
	if p.ProfileID == "" {
		return errors.New("profile required")
	}
	switch p.Service {
	case "pppoe", "ipoe", "hotspot", "ftth":
		return nil
	default:
		return errors.New("unknown service type")
	}
}
