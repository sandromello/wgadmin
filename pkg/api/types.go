package api

import (
	"net"
	"time"
)

// KeyLen is the expected key length for a WireGuard key.
const KeyLen = 32 // wgh.KeyLen

// A Key is a public, private, or pre-shared secret key.  The Key constructor
// functions in this package can be used to create Keys suitable for each of
// these applications.
type Key [KeyLen]byte

// Metadata common attributes to all objects
type Metadata struct {
	UID string `json:"uid"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// PeerStatus indicates the status of a given peer
type PeerStatus string

const (
	// PeerStatusInitial indicates that the peer is registered and a configuration
	// is available for download.
	PeerStatusInitial PeerStatus = ""
	// PeerStatusActive indicates that peer is configured on the wireguard server
	// and could be used to establish a connection with it.
	PeerStatusActive PeerStatus = "active"
	// PeerStatusBlocked indicates that the peer is unable to download configuration
	// or establish connection with the server.
	PeerStatusBlocked PeerStatus = "blocked"
)

// PeerExpireActionType indicate what to do when the peer is expired
type PeerExpireActionType string

const (
	// PeerExpireActionDefault is the default mode and peers will never expire
	PeerExpireActionDefault PeerExpireActionType = ""
	// PeerExpireActionReset will expire peers after a specified time
	// the client will need to eventually ask for a new client config.
	// The duration is calculated using the .metadata.createdAt attribute of a peer
	PeerExpireActionReset PeerExpireActionType = "reset"
	// PeerExpireActionBlock it will remove the peer without expiring it.
	// The client will need to unblock its peer from time configured basis.
	// The duration is calculated using the .metadata.updatedAt attribute of a peer
	PeerExpireActionBlock PeerExpireActionType = "block"
)

// WireguardServerConfig represents the main config server of a wireguard server
type WireguardServerConfig struct {
	Metadata `json:",inline"`

	Address        string   `json:"address"`
	ListenPort     int      `json:"listenPort"`
	PrivateKey     *Key     `json:"privateKey"`
	PostUp         []string `json:"postUp"`
	PostDown       []string `json:"postDown"`
	PublicEndpoint string   `json:"publicEndpoint"`

	// Peers from this server will inheret this value
	PeerExpireAction PeerExpireActionType `json:"peerExpireAction"`

	// Only peers with status active
	ActivePeers []Peer `json:"peers"` // TODO: deprecate in flavor of peer selector
	// Select peers matching labels of a giving peer object
	// PeerSelector map[string]string `json:"selector"`
}

// Peer is a section of peer in a wg server config file
type Peer struct {
	Metadata `json:",inline"`

	PublicKey      *Key                 `json:"publicKey"`
	AllowedIPs     net.IPNet            `json:"allowedIPs"`
	SecretValue    string               `json:"secretValue"`
	ExpireAction   PeerExpireActionType `json:"peerExpireAction"`
	ExpireDuration time.Duration        `json:"expireDuration"`

	Status PeerStatus `json:"status"`
}

// WireguardClientConfig represents a wireguard client config
type WireguardClientConfig struct {
	UID string `json:"uid"`

	InterfaceClientConfig InterfaceClientConfig `json:"interface"`
	PeerClientConfig      PeerClientConfig      `json:"peer"`
	// PgpMessage
}

// InterfaceClientConfig represents a Interface section on a client wireguard config
// https://git.zx2c4.com/WireGuard/about/src/tools/man/wg.8
type InterfaceClientConfig struct {
	PrivateKey *Key       `json:"privateKey"`
	Address    *net.IPNet `json:"address"`
	DNS        []net.IP   `json:"dns"`
}

// PeerClientConfig represents a Peer section on a client wireguard config
// https://git.zx2c4.com/WireGuard/about/src/tools/man/wg.8
type PeerClientConfig struct {
	PublicKey           string      `json:"publicKey"`
	AllowedIPs          []net.IPNet `json:"allowedIPs"`
	Endpoint            string      `json:"endpoint"`
	PersistentKeepAlive int         `json:"persistentKeepAlive"`
}

// // PGPPublicKey represents a PGP Public Key used to encrypt the wireguard client config
// type PGPPublicKey struct {
// 	UID string `json:"uid"`

// 	Name string `json:"name"`
// 	Key  string `json:"key"`
// }
