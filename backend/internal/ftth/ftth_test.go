package ftth_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/ftth"
)

func TestDiagnose(t *testing.T) {
	ok := ftth.Diagnose(ftth.ONU{Serial: "s", Status: "online", RxPower: -19, Temperature: 45})
	if ok.Level != "ok" {
		t.Fatalf("%+v", ok)
	}
	deg := ftth.Diagnose(ftth.ONU{Serial: "s", Status: "online", RxPower: -26})
	if deg.Level != "degraded" {
		t.Fatalf("%+v", deg)
	}
	down := ftth.Diagnose(ftth.ONU{Serial: "s", Status: "online", RxPower: -30})
	if down.Level != "down" {
		t.Fatalf("%+v", down)
	}
	gasp := ftth.Diagnose(ftth.ONU{Serial: "s", Status: "dyinggasp"})
	if gasp.Level != "down" {
		t.Fatalf("%+v", gasp)
	}
}

func TestPlans(t *testing.T) {
	p := ftth.Profile{Vendor: "ZTE", Name: "FTTH-50M", DownMbps: 50, UpMbps: 20}
	pr := ftth.Provision("SN1", p, 100)
	if pr.Action != "provision" || pr.Profile.VLAN != 100 {
		t.Fatalf("%+v", pr)
	}
	if a := ftth.Authorize("SN1", p); a.Action != "authorize" {
		t.Fatalf("%+v", a)
	}
	if d := ftth.Deprovision("SN1"); d.Action != "deprovision" {
		t.Fatalf("%+v", d)
	}
}
