package api

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"text/template"

	"golang.org/x/crypto/curve25519"
)

// PublicKey computes a public key from the private key k.
//
// PublicKey should only be called when k is a private key.
func (k Key) PublicKey() Key {
	var (
		pub  [KeyLen]byte
		priv = [KeyLen]byte(k)
	)

	// ScalarBaseMult uses the correct base value per https://cr.yp.to/ecdh.html,
	// so no need to specify it.
	curve25519.ScalarBaseMult(&pub, &priv)
	return Key(pub)
}

// String returns the base64-encoded string representation of a Key.
//
// ParseKey can be used to produce a new Key from this string.
func (k Key) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

// MarshalJSON serialize a Key struct to a base64 value
func (k *Key) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// UnmarshalJSON deserialize a base64 key to a Key struct
func (k *Key) UnmarshalJSON(data []byte) error {
	var base64Key string
	if err := json.Unmarshal(data, &base64Key); err != nil {
		return err
	}
	key, err := ParseKey(string(base64Key))
	if err == nil {
		copy(k[:], key[:])
	}
	return err
}

// ParseDNSToComma parses the DNS config to a comma for each entry
func (i InterfaceClientConfig) ParseDNSToComma() string {
	var dnss []string
	for _, d := range i.DNS {
		dnss = append(dnss, d.String())
	}
	return strings.Join(dnss, ", ")
}

// ParseAllowedIPsToComma parses the AllowedIPs config to a comma for each entry
func (p PeerClientConfig) ParseAllowedIPsToComma() string {
	var ips []string
	for _, ip := range p.AllowedIPs {
		ips = append(ips, ip.String())
	}
	return strings.Join(ips, ", ")
}

// MarshalJSON serializes to an encoded format
// func (k *Key) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(k.String())
// }

// NewKey creates a Key from an existing byte slice.  The byte slice must be
// exactly 32 bytes in length.
func NewKey(b []byte) (Key, error) {
	if len(b) != KeyLen {
		return Key{}, fmt.Errorf("incorrect key size: %d", len(b))
	}

	var k Key
	copy(k[:], b)

	return k, nil
}

// ParseKey parses a Key from a base64-encoded string, as produced by the
// Key.String method.
func ParseKey(s string) (Key, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Key{}, fmt.Errorf("failed to parse base64-encoded key: %v", err)
	}

	return NewKey(b)
}

// GenerateKey generates a Key suitable for use as a pre-shared secret key from
// a cryptographically safe source.
//
// The output Key should not be used as a private key; use GeneratePrivateKey
// instead.
func GenerateKey() (Key, error) {
	b := make([]byte, KeyLen)
	if _, err := rand.Read(b); err != nil {
		return Key{}, fmt.Errorf("failed to read random bytes: %v", err)
	}

	return NewKey(b)
}

// GeneratePrivateKey generates a Key suitable for use as a private key from a
// cryptographically safe source.
func GeneratePrivateKey() (Key, error) {
	key, err := GenerateKey()
	if err != nil {
		return Key{}, err
	}

	// Modify random bytes using algorithm described at:
	// https://cr.yp.to/ecdh.html.
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64

	return key, nil
}

// ParseAllowedIPs will parse all CIDR and return a []net.IPNet
// if a given CIDR is in wrong format will be ignored
func ParseAllowedIPs(cidrs ...string) []net.IPNet {
	var allowedIPs []net.IPNet
	for _, cidr := range cidrs {
		ipnet := ParseCIDR(cidr)
		if ipnet == nil {
			continue
		}
		allowedIPs = append(allowedIPs, *ipnet)
	}
	return allowedIPs
}

// ParseCIDR tries to parse a ipv4 or ipv6, returns nil otherwise
func ParseCIDR(s string) *net.IPNet {
	_, ipnet, _ := net.ParseCIDR(s)
	return ipnet
}

// HandleTemplates parse template and write to file
func HandleTemplates(text string, file io.Writer, data interface{}) error {
	t, err := template.New("").Parse(text)
	if err != nil {
		return fmt.Errorf("failed parsing template, err =%v", err)
	}
	err = t.Execute(file, data)
	if err != nil {
		return fmt.Errorf("failed executing template, err=%v", err)
	}
	return nil
}

// WriteToIniFile parse the wireguard server config template and write to a given file
func (w *WireguardServerConfig) WriteToIniFile(filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return HandleTemplates(string(templateWireguardServerConfig), f, w)
}

// ParseWireguardServerConfigTemplate parse to []byte the wireguard server config template
func (w *WireguardServerConfig) ParseWireguardServerConfigTemplate() ([]byte, error) {
	var buf bytes.Buffer
	if err := HandleTemplates(string(templateWireguardServerConfig), &buf, w); err != nil {
		return nil, fmt.Errorf("failed parsing template: %v", err)
	}
	return buf.Bytes(), nil
}

// ParseWireguardClientConfigTemplate parse to []byte the wireguard client config template
func ParseWireguardClientConfigTemplate(obj map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := HandleTemplates(string(templateWireguardClientConfig), &buf, obj); err != nil {
		return nil, fmt.Errorf("failed parsing template: %v", err)
	}
	return buf.Bytes(), nil
}
