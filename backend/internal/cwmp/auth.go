package cwmp

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// HTTP auth for the ACS endpoint (RFC 7617 Basic + RFC 7616 Digest MD5).
// Users maps username -> password. Empty map = open (lab only, logged).

type authState struct {
	mu    sync.Mutex
	nonce map[string]bool
}

var nonces = &authState{nonce: map[string]bool{}}

func newNonce() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	n := hex.EncodeToString(b[:])
	nonces.mu.Lock()
	nonces.nonce[n] = true
	nonces.mu.Unlock()
	return n
}

func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// MD5Hex is exported for tests and tooling.
func MD5Hex(s string) string { return md5hex(s) }

// CheckAuth validates Authorization (Basic or Digest, qop=auth, MD5).
// Returns true when authorized OR when no users are configured.
func CheckAuth(users map[string]string, r *http.Request) bool {
	if len(users) == 0 {
		return true
	}
	ah := r.Header.Get("Authorization")
	if strings.HasPrefix(ah, "Basic ") {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ah, "Basic "))
		if err != nil {
			return false
		}
		u, p, ok := strings.Cut(string(raw), ":")
		if !ok {
			return false
		}
		want, ok := users[u]
		return ok && subtleEq(want, p)
	}
	if strings.HasPrefix(ah, "Digest ") {
		f := parseDigest(strings.TrimPrefix(ah, "Digest "))
		pw, ok := users[f["username"]]
		if !ok {
			return false
		}
		nonces.mu.Lock()
		_, fresh := nonces.nonce[f["nonce"]]
		nonces.mu.Unlock()
		if !fresh {
			return false
		}
		ha1 := md5hex(f["username"] + ":" + f["realm"] + ":" + pw)
		ha2 := md5hex(r.Method + ":" + f["uri"])
		want := md5hex(ha1 + ":" + f["nonce"] + ":" + f["nc"] + ":" + f["cnonce"] + ":" + f["qop"] + ":" + ha2)
		return subtleEq(want, f["response"])
	}
	return false
}

func subtleEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	eq := true
	for i := range a {
		if a[i] != b[i] {
			eq = false
		}
	}
	return eq
}

func parseDigest(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		out[k] = strings.Trim(v, `"`)
	}
	return out
}

// Challenge issues a Digest challenge for the ACS realm.
func Challenge(w http.ResponseWriter, realm string) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm="%s", qop="auth", nonce="%s", algorithm=MD5`, realm, newNonce()))
	w.WriteHeader(http.StatusUnauthorized)
}
