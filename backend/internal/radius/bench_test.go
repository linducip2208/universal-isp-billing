package radius_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/radius"
)

func BenchmarkCodec(b *testing.B) {
	p := &radius.Packet{Code: radius.CodeAccessRequest, Identifier: 7,
		Attrs: []radius.Attr{{Type: 1, Value: []byte("alice")}, {Type: 2, Value: []byte("secret-pass")}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := radius.Encode(p, "s3cret", p.Authenticator)
		if _, err := radius.Decode(raw); err != nil {
			b.Fatal(err)
		}
	}
}
