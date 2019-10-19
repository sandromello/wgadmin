package api

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	w := &WireguardServerConfig{
		UID:        "foo",
		Address:    ParseCIDR("10.100.0.10/32"),
		ListenPort: 51820,
		PrivateKey: &pk,
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
	var buf bytes.Buffer
	if err := HandleTemplates(string(templateWireguardServerConfig), &buf, w); err != nil {
		t.Fatalf("failed handling template: %v", err)
	}
	if diff := cmp.Diff(string(expOutput), buf.String()); diff != "" {
		t.Fatalf("unexpected ini output (-want +got):\n%s", diff)
	}
}

func TestWireguardServerPeersConfigToIni(t *testing.T) {
	expOutput := []byte(`[Peer]
PublicKey = 8AO7DXtcu//EolOd9zevkU6Rro0DDgCbnjXm4OUWcWs=
AllowedIPs = 192.168.180.2/32

[Peer]
PublicKey = b/p252rtUdFg7Z7ENKsjguhBYfjsplzvWKCCWGsOCgo=
AllowedIPs = 192.168.180.3/32

[Peer]
PublicKey = T7WIUxjK4koRiKles/5dNUs5naOJtJf4Oq8m6IeVaxM=
AllowedIPs = 192.168.180.4/32

`)
	// wg genkey | wg pubkey
	pubkey01, _ := ParseKey("8AO7DXtcu//EolOd9zevkU6Rro0DDgCbnjXm4OUWcWs=")
	pubkey02, _ := ParseKey("b/p252rtUdFg7Z7ENKsjguhBYfjsplzvWKCCWGsOCgo=")
	pubkey03, _ := ParseKey("T7WIUxjK4koRiKles/5dNUs5naOJtJf4Oq8m6IeVaxM=")
	w := &WireguardServerConfig{
		ActivePeers: []Peer{
			{
				PublicKey:  &pubkey01,
				AllowedIPs: *ParseCIDR("192.168.180.2/32"),
			},
			{
				PublicKey:  &pubkey02,
				AllowedIPs: *ParseCIDR("192.168.180.3/32"),
			},
			{
				PublicKey:  &pubkey03,
				AllowedIPs: *ParseCIDR("192.168.180.4/32"),
			},
		},
	}
	var buf bytes.Buffer
	if err := HandleTemplates(string(templateWireguardServerPeersConfig), &buf, w); err != nil {
		t.Fatalf("failed handling template: %v", err)
	}
	if diff := cmp.Diff(string(expOutput), buf.String()); diff != "" {
		t.Fatalf("unexpected ini output (-want +got):\n%s", diff)
	}
}

func TestWireguardClientConfigToIni(t *testing.T) {
	expOutput := []byte(`[Interface]
PrivateKey = +P5vXpg6yPdq5mG1KNmSamFPKvzbBEJ6OqYYidwKREo=
Address    = 192.168.180.2/32
DNS        = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey  = Xn9vjTRVlfm2nxVTMATZy73EJWRYWv7db2z13o2e5R4=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint   = wg-dev.vpn.domain.tld:51820

PersistentKeepalive = 25
`)
	// wg genkey
	privkey, _ := ParseKey("+P5vXpg6yPdq5mG1KNmSamFPKvzbBEJ6OqYYidwKREo=")
	w := &WireguardClientConfig{
		UID: "foo",
		InterfaceClientConfig: InterfaceClientConfig{
			PrivateKey: &privkey,
			Address:    ParseCIDR("192.168.180.2/32"),
			DNS:        []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("8.8.8.8")},
		},
		PeerClientConfig: PeerClientConfig{
			PublicKey:           privkey.PublicKey().String(),
			AllowedIPs:          ParseAllowedIPs("0.0.0.0/0", "::/0"),
			Endpoint:            "wg-dev.vpn.domain.tld:51820",
			PersistentKeepAlive: 25,
		},
	}

	var buf bytes.Buffer
	if err := HandleTemplates(string(templateWireguardClientConfig), &buf, w); err != nil {
		t.Fatalf("failed handling template: %v", err)
	}
	if diff := cmp.Diff(string(expOutput), buf.String()); diff != "" {
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
