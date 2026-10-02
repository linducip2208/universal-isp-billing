package radius_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/radius"
)

func testCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "radsec-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		IsCA:         true, BasicConstraintsValid: true,
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(
		pemBlock("CERTIFICATE", der),
		pemBlock("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key)))
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func pemBlock(typ string, der []byte) []byte {
	enc := base64.StdEncoding.EncodeToString(der)
	var sb strings.Builder
	sb.WriteString("-----BEGIN " + typ + "-----\n")
	for i := 0; i < len(enc); i += 64 {
		end := i + 64
		if end > len(enc) {
			end = len(enc)
		}
		sb.WriteString(enc[i:end] + "\n")
	}
	sb.WriteString("-----END " + typ + "-----\n")
	return []byte(sb.String())
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func TestRadSecLoopback(t *testing.T) {
	cert := testCert(t)
	srv := &radius.Server{
		Secret: "s3cret",
		Verify: func(u, p string) (map[string]string, bool) {
			return map[string]string{}, u == "bob" && p == "pw2"
		},
		Sessions: radius.NewSessionStore(),
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("no tcp")
	}
	addr := ln.Addr().String()
	ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = srv.ServeTLS(ctx, addr, &tls.Config{Certificates: []tls.Certificate{cert}})
	}()
	time.Sleep(150 * time.Millisecond)
	var ra [16]byte
	_, _ = rand.Read(ra[:])
	ct, err := radius.EncryptPAP("s3cret", ra, "pw2")
	if err != nil {
		t.Fatal(err)
	}
	got, err := radius.SendTLS(addr, &tls.Config{InsecureSkipVerify: true},
		&radius.Packet{Code: radius.CodeAccessRequest, Identifier: 5, Authenticator: ra,
			Attrs: []radius.Attr{{Type: 1, Value: []byte("bob")}, {Type: 2, Value: ct}}}, 3*time.Second)
	if err != nil {
		t.Fatalf("radsec: %v", err)
	}
	if got.Code != radius.CodeAccessAccept {
		t.Fatalf("want accept, got %d", got.Code)
	}
}
