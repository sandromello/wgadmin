package api

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sandromello/wgadmin/pkg/util"
	"golang.org/x/crypto/curve25519"
)

func (d ServerDaemon) GetUnitName() string    { return d.UnitName }
func (d ServerDaemon) GetSystemdPath() string { return d.SystemdPath }
func (d PeerDaemon) GetUnitName() string      { return d.UnitName }
func (d PeerDaemon) GetSystemdPath() string   { return d.SystemdPath }

// GetWireguardConfigFile returns the path to the wireguard config file
func (d ServerConfig) GetWireguardConfigFile() string {
	return filepath.Join(d.ServerDaemon.ConfigPath, d.ServerDaemon.ConfigFile)
}

// UnmarshalJSON deserialize a time.Duration
func (d *Duration) UnmarshalJSON(data []byte) error {
	unquoted, err := strconv.Unquote(string(data))
	if err != nil {
		return nil
	}
	duration, err := time.ParseDuration(unquoted)
	d2 := Duration(duration)
	*d = d2
	return err
}

// PublicKey computes a public key from the private key k.
// It should only be called when k is a private key.
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

// PublicKeyString return the public key from a peer,
// returns an empty string if it's empty
func (p *Peer) PublicKeyString() string {
	pubkey := p.GetPublicKey()
	if pubkey == nil {
		return ""
	}
	return pubkey.String()
}

// GetPublicKey returns a persistent public key, otherwise returns a volatile one
func (p *Peer) GetPublicKey() *Key {
	if p.Spec.PersistentPublicKey != nil {
		return p.Spec.PersistentPublicKey
	}
	return p.Status.PublicKey
}

// GetStatus get the status of a peer
func (p *Peer) GetStatus() PeerPhase {
	if p.Spec.Blocked {
		return PeerBlocked
	}
	if p.GetPublicKey() == nil {
		return PeerPending
	}
	if p.Spec.ExpireAction != PeerExpireActionDefault &&
		p.Spec.PersistentPublicKey == nil &&
		p.IsExpired() {
		return PeerExpired
	}
	return PeerActive
}

// GetServer retrieves the wireguard server which this peer belongs to
func (p *Peer) GetServer() string {
	return strings.Split(p.UID, "/")[0]
}

// ShouldAutoLock verify if a peer should be locked
func (p *Peer) ShouldAutoLock() bool {
	return p.Spec.ExpireAction != PeerExpireActionDefault && p.IsExpired()
}

// IsExpired check if the peer is expired
func (p *Peer) IsExpired() bool {
	return p.GetExpirationDuration() <= 0
}

// GetExpirationDuration retrieves the expiration of a given peer
func (p *Peer) GetExpirationDuration() time.Duration {
	if p.GetPublicKey() == nil {
		return time.Duration(0)
	}
	var t time.Time
	switch p.Spec.ExpireAction {
	case PeerExpireActionBlock:
		t, _ = time.Parse(time.RFC3339, p.UpdatedAt)
	case PeerExpireActionReset:
		t, _ = time.Parse(time.RFC3339, p.CreatedAt)
	}
	return util.RoundTime((p.ParseExpireDuration() - time.Now().UTC().Sub(t)), time.Second)
}

// ParseExpireDuration parse the duration of the given expiration time
func (p *Peer) ParseExpireDuration() time.Duration {
	d, _ := time.ParseDuration(p.Spec.ExpireDuration)
	return d
}

// ParseAllowedIPs parse the allowed ip's and return a net.IP.
// It will return nil if it's in wrong format
func (p *Peer) ParseAllowedIPs() net.IP {
	ipaddr, _, _ := net.ParseCIDR(p.Spec.AllowedIPs)
	return ipaddr
}

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

// ParseCIDR tries to parse a ipv4 or ipv6, returns nil otherwise
func ParseCIDR(s string) *net.IPNet {
	_, ipnet, _ := net.ParseCIDR(s)
	return ipnet
}

// HandleTemplates parse template and write to file
func HandleTemplates(text string, file io.Writer, data interface{}) error {
	t, err := template.New("").Parse(text)
	if err != nil {
		return fmt.Errorf("failed parsing template, err=%v", err)
	}
	err = t.Execute(file, data)
	if err != nil {
		return fmt.Errorf("failed executing template, err=%v", err)
	}
	return nil
}

// ParseWireguardServerConfigTemplate parse to []byte the wireguard server config template
func (w *WireguardServerConfig) ParseWireguardServerConfigTemplate(cipherKey string) ([]byte, error) {
	var buf bytes.Buffer
	privKey, err := w.DecryptPrivateKey(cipherKey)
	if err != nil {
		return nil, err
	}
	if err := HandleTemplates(string(templateWireguardServerConfig), &buf, map[string]interface{}{
		"PrivateKey": privKey.String(),
		"Address":    w.Address,
		"ListenPort": w.ListenPort,
		"PostUp":     w.PostUp,
		"PostDown":   w.PostDown,
	}); err != nil {
		return nil, fmt.Errorf("failed parsing template: %v", err)
	}
	return buf.Bytes(), nil
}

// DecryptPrivateKey decrypt the private key using the given cipher key
func (w *WireguardServerConfig) DecryptPrivateKey(cipherKey string) (Key, error) {
	cipher, err := util.NewAESCipherKey(cipherKey)
	if err != nil {
		return Key{}, fmt.Errorf("failed creating cipher key: %v", err)
	}
	privKeyEncoded, err := cipher.DecryptMessage(w.EncryptedPrivateKey)
	if err != nil {
		return Key{}, fmt.Errorf("failed decrypting private key: %v", err)
	}
	return ParseKey(privKeyEncoded)
}

// ParseWireguardClientConfigTemplate parse to []byte the wireguard client config template
func ParseWireguardClientConfigTemplate(obj map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := HandleTemplates(string(templateWireguardClientConfig), &buf, obj); err != nil {
		return nil, fmt.Errorf("failed parsing template: %v", err)
	}
	return buf.Bytes(), nil
}

// SortPeerByUID sorts peers by uid.
type SortPeerByUID []Peer

func (a SortPeerByUID) Len() int           { return len(a) }
func (a SortPeerByUID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortPeerByUID) Less(i, j int) bool { return a[i].UID < a[j].UID }
