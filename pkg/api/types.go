package api

import (
	"net"
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

// PeerPhase indicates in which state a peer is
type PeerPhase string

const (
	// PeerPending indicates that the peer is registered
	// and awaiting for download.
	PeerPending PeerPhase = "pending"
	// PeerBlocked indicates that the peer is unable to download
	// a configuration or establish connection with the server.
	PeerBlocked PeerPhase = "blocked"
	// PeerActive indicates that peer is configured on the wireguard server
	// and could be used to establish a connection with it.
	PeerActive PeerPhase = "active"
	// PeerExpired indicates that the peer is configured to expire and
	// its lifespan has expired
	PeerExpired PeerPhase = "expired"
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

	Address             string   `json:"address"`
	ListenPort          int      `json:"listenPort"`
	EncryptedPrivateKey string   `json:"encryptedPrivateKey"`
	PrivateKey          *Key     `json:"privateKey"` // TODO: remove in flavor of EncryptedPrivateKey
	PublicKey           *Key     `json:"publicKey"`
	PostUp              []string `json:"postUp"`
	PostDown            []string `json:"postDown"`
	PublicEndpoint      string   `json:"publicEndpoint"`

	// Peers from this server will inheret this value
	PeerExpireAction PeerExpireActionType `json:"peerExpireAction"`

	// Only peers with status active
	ActivePeers []Peer `json:"peers"` // TODO: deprecate in flavor of peer selector
}

// Peer is a section of peer in a wg server config file
type Peer struct {
	Metadata `json:"metadata"`

	Spec   PeerSpec   `json:"spec"`
	Status PeerStatus `json:"status"`
}

// PeerSpec main configuration of a peer
type PeerSpec struct {
	PublicKey      *Key                 `json:"publicKey"`
	AllowedIPs     string               `json:"allowedIPs"`
	ExpireAction   PeerExpireActionType `json:"expireAction"`
	ExpireDuration string               `json:"expireDuration"`
	Blocked        bool                 `json:"blocked"`
}

// PeerStatus hold status of a peer
type PeerStatus struct {
	Phase       PeerPhase `json:"phase"`
	SecretValue string    `json:"secretValue"`
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
