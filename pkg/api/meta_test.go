package api

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sandromello/wgadmin/pkg/util"
	"golang.org/x/crypto/curve25519"
)

func TestWireguardServerConfigToIni(t *testing.T) {
	// PrivateKey was generated using: wg genkey
	expOutput := []byte(`[Interface]
Address    = 10.100.0.10/32
ListenPort = 51820
PrivateKey = GNyaar5SFUOf3emHLP+dhyTTKT6zXlmkZB0bg2uuFHQ=

PostUp = ip link set mtu 1500 dev ens4
PostUp = ip link set mtu 1500 dev %i
PostUp = sysctl -w net.ipv4.ip_forward=1

PostDown = sysctl -w net.ipv4.ip_forward=0
PostDown = iptables -D FORWARD -o %i -j ACCEPT
PostDown = iptables -D FORWARD -i %i -j ACCEPT
`)
	pk, err := ParseKey("GNyaar5SFUOf3emHLP+dhyTTKT6zXlmkZB0bg2uuFHQ=")
	if err != nil {
		t.Fatalf("failed parsing key: %v", err)
	}
	cipherKey, err := util.NewAESCipherKey("")
	if err != nil {
		t.Fatalf("failed generating cipher key: %v", err)
	}
	encPK, err := cipherKey.EncryptMessage(pk.String())
	if err != nil {
		t.Fatalf("failed encrypting private key: %v", err)
	}
	w := &WireguardServerConfig{
		Metadata: Metadata{
			UID: "foo",
		},
		Address:             ParseCIDR("10.100.0.10/32").String(),
		ListenPort:          51820,
		EncryptedPrivateKey: encPK,
		PostUp: []string{
			"ip link set mtu 1500 dev ens4",
			"ip link set mtu 1500 dev %i",
			"sysctl -w net.ipv4.ip_forward=1",
		},
		PostDown: []string{
			"sysctl -w net.ipv4.ip_forward=0",
			"iptables -D FORWARD -o %i -j ACCEPT",
			"iptables -D FORWARD -i %i -j ACCEPT",
		},
	}
	configData, err := w.ParseWireguardServerConfigTemplate(cipherKey.String())
	if err != nil {
		t.Fatalf("failed handling template: %v", err)
	}
	if diff := cmp.Diff(string(expOutput), string(configData)); diff != "" {
		t.Fatalf("unexpected ini output (-want +got):\n%s", diff)
	}
}

func TestPreparedKeys(t *testing.T) {
	// Keys generated via "wg genkey" and "wg pubkey" for comparison
	// with this Go implementation.
	const (
		private = "GHuMwljFfqd2a7cs6BaUOmHflK23zME8VNvC5B37S3k="
		public  = "aPxGwq8zERHQ3Q1cOZFdJ+cvJX5Ka4mLN38AyYKYF10="
	)

	priv, err := ParseKey(private)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	if diff := cmp.Diff(private, priv.String()); diff != "" {
		t.Fatalf("unexpected private key (-want +got):\n%s", diff)
	}

	pub := priv.PublicKey()
	if diff := cmp.Diff(public, pub.String()); diff != "" {
		t.Fatalf("unexpected public key (-want +got):\n%s", diff)
	}
}

func TestKeyExchange(t *testing.T) {
	privA, pubA := mustKeyPair()
	privB, pubB := mustKeyPair()

	// Perform ECDH key exhange: https://cr.yp.to/ecdh.html.
	var sharedA, sharedB [32]byte
	curve25519.ScalarMult(&sharedA, privA, pubB)
	curve25519.ScalarMult(&sharedB, privB, pubA)

	if diff := cmp.Diff(sharedA, sharedB); diff != "" {
		t.Fatalf("unexpected shared secret (-want +got):\n%s", diff)
	}
}

func TestBadKeys(t *testing.T) {
	// Adapt to fit the signature used in the test table.
	parseKey := func(b []byte) (Key, error) {
		return ParseKey(string(b))
	}

	tests := []struct {
		name string
		b    []byte
		fn   func(b []byte) (Key, error)
	}{
		{
			name: "bad base64",
			b:    []byte("xxx"),
			fn:   parseKey,
		},
		{
			name: "short base64",
			b:    []byte("aGVsbG8="),
			fn:   parseKey,
		},
		{
			name: "short key",
			b:    []byte("xxx"),
			fn:   NewKey,
		},
		{
			name: "long base64",
			b:    []byte("ZGVhZGJlZWZkZWFkYmVlZmRlYWRiZWVmZGVhZGJlZWZkZWFkYmVlZg=="),
			fn:   parseKey,
		},
		{
			name: "long bytes",
			b:    bytes.Repeat([]byte{0xff}, 40),
			fn:   NewKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn(tt.b)
			if err == nil {
				t.Fatal("expected an error, but none occurred")
			}

			t.Logf("OK error: %v", err)
		})
	}
}

// func TestSerializeWireguardServerConfig(t *testing.T) {
// 	priv, err := GeneratePrivateKey()
// 	if err != nil {
// 		panicf("failed to generate private key: %v", err)
// 	}
// 	wgconf := WireguardServerConfig{
// 		UID:        "foo",
// 		Address:    "192.168.150.1/32",
// 		ListenPort: 5180,
// 		PrivateKey: &priv,
// 		PostUp:     []string{"postup-commands"},
// 		PostDown:   []string{"postdown-commands"},
// 	}
// 	data, err := json.Marshal(wgconf)
// 	if err != nil {
// 		t.Fatalf("failed serializing: %v", err)
// 	}
// 	t.Log(string(data))

// }

func mustKeyPair() (private, public *[32]byte) {
	priv, err := GeneratePrivateKey()
	if err != nil {
		panicf("failed to generate private key: %v", err)
	}

	return keyPtr(priv), keyPtr(priv.PublicKey())
}

func keyPtr(k Key) *[32]byte {
	b32 := [32]byte(k)
	return &b32
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
