package cwmp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/cwmp"
)

func TestBasicAuth(t *testing.T) {
	users := map[string]string{"cpe": "s3cret"}
	ok := httptest.NewRequest("POST", "/acs", nil)
	ok.SetBasicAuth("cpe", "s3cret")
	if !cwmp.CheckAuth(users, ok) {
		t.Fatal("valid basic must pass")
	}
	bad := httptest.NewRequest("POST", "/acs", nil)
	bad.SetBasicAuth("cpe", "wrong")
	if cwmp.CheckAuth(users, bad) {
		t.Fatal("wrong password must fail")
	}
	none := httptest.NewRequest("POST", "/acs", nil)
	if cwmp.CheckAuth(users, none) {
		t.Fatal("missing auth must fail")
	}
	if !cwmp.CheckAuth(map[string]string{}, none) {
		t.Fatal("empty users = open lab mode")
	}
}

func TestDigestAuth(t *testing.T) {
	users := map[string]string{"cpe": "s3cret"}
	// get challenge nonce
	rec := httptest.NewRecorder()
	cwmp.Challenge(rec, "acs")
	wh := rec.Header().Get("WWW-Authenticate")
	nonce := ""
	for _, p := range strings.Split(wh, ",") {
		if strings.Contains(p, "nonce=") {
			nonce = strings.Trim(strings.Split(p, "=")[1], `" `)
		}
	}
	if nonce == "" {
		t.Fatal("no nonce in challenge")
	}
	mk := func(user, pass, nc string) *http.Request {
		r := httptest.NewRequest("POST", "/acs", nil)
		resp := digestResp(user, "acs", pass, nonce, nc, "abcdef", "/acs")
		r.Header.Set("Authorization", "Digest "+resp)
		return r
	}
	if !cwmp.CheckAuth(users, mk("cpe", "s3cret", "00000001")) {
		t.Fatal("valid digest must pass")
	}
	if cwmp.CheckAuth(users, mk("cpe", "wrong", "00000002")) {
		t.Fatal("wrong password must fail")
	}
}

func digestResp(user, realm, pass, nonce, nc, cnonce, uri string) string {
	h := func(s string) string { return cwmp.MD5Hex(s) }
	ha1 := h(user + ":" + realm + ":" + pass)
	ha2 := h("POST:" + uri)
	resp := h(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":auth:" + ha2)
	return `username="` + user + `", realm="` + realm + `", nonce="` + nonce + `", uri="` + uri +
		`", qop=auth, nc=` + nc + `, cnonce="` + cnonce + `", response="` + resp + `"`
}
