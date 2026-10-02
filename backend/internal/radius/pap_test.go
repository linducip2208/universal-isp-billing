package radius_test

import (
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/radius"
)

func TestPAPRoundTrip(t *testing.T) {
	var auth [16]byte
	for i := range auth {
		auth[i] = byte(i)
	}
	for _, pw := range []string{"a", "pw", "sixteen-chars!!", "a much longer password over sixteen bytes", strings.Repeat("x", 128)} {
		ct, err := radius.EncryptPAP("s3cret", auth, pw)
		if err != nil {
			t.Fatalf("%q: %v", pw, err)
		}
		if len(ct)%16 != 0 {
			t.Fatalf("%q: bad length %d", pw, len(ct))
		}
		back, err := radius.DecryptPAP("s3cret", auth, ct)
		if err != nil || back != pw {
			t.Fatalf("%q -> %q err=%v", pw, back, err)
		}
		// wrong secret must not recover
		if bad, _ := radius.DecryptPAP("other", auth, ct); bad == pw {
			t.Fatalf("%q recovered with wrong secret", pw)
		}
	}
	if _, err := radius.EncryptPAP("s", auth, ""); err == nil {
		t.Fatal("empty must fail")
	}
	if _, err := radius.EncryptPAP("s", auth, strings.Repeat("x", 129)); err == nil {
		t.Fatal(">128 must fail")
	}
	if _, err := radius.DecryptPAP("s", auth, []byte{1, 2, 3}); err == nil {
		t.Fatal("unaligned must fail")
	}
}
