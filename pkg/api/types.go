package api

import (
	"net"
	"time"
)

const PeerDefaultMTU string = "1280"

// WebApp holds information about the webapp server
type WebApp struct {
	HTTPPort                     string      `json:"httpPort"`
	AllowedDomains               []string    `json:"allowedDomains"`
	PageConfig                   *PageConfig `json:"pageConfig"`
	TLSKeyFile                   string      `json:"tlsKeyFile"`
	TLSCertFile                  string      `json:"tlsCertFile"`
	GoogleApplicationCredentials string      `json:"googleApplicationCredentials"`
	GCSBucketName                string      `json:"gcsBucketName"`
}

// PageConfig is used to configure the content of the webapp
type PageConfig struct {
	FaviconURL        string `json:"faviconURL"`
	LogoURL           string `json:"logoURL"`
	ThemeCSSURL       string `json:"themeCSSURL"`
	GoogleClientID    string `json:"googleClientID"`
	GoogleRedirectURI string `json:"googleRedirectURI"`
	TemplatePath      string `json:"templatePath"`
	Title             string `json:"title"`
	NavBarLink        string `json:"navbarLink"`
}

// Duration a custom time.Duration
type Duration time.Duration

// Daemon represents a systemd unit
type Daemon interface {
	GetUnitName() string
	GetSystemdPath() string
}

// ServerConfig is all the required configuration to run the
// daemons that sync peers and servers
type ServerConfig struct {
	Name         string       `json:"name"`
	BucketName   string       `json:"bucketName"`
	ServerDaemon ServerDaemon `json:"server"`
	PeerDaemon   PeerDaemon   `json:"peer"`
}

// PeerDaemon is a configuration to tell how to synchronize and configure peers
type PeerDaemon struct {
	UnitName      string   `json:"unitName"`
	SystemdPath   string   `json:"systemdPath"`
	SyncTime      Duration `json:"syncTime"`
	InterfaceName string   `json:"interfaceName"`
}

// ServerDaemon is a configuration to tell how to synchronize and configure a server
type ServerDaemon struct {
	UnitName    string   `json:"unitName"`
	SystemdPath string   `json:"systemdPath"`
	SyncTime    Duration `json:"syncTime"`
	CipherKey   string   `json:"cipherKey"`
	ConfigPath  string   `json:"configPath"`
	ConfigFile  string   `json:"configFile"`
}

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
	PeerPending PeerPhase = "pending download"
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
	PublicKey           *Key     `json:"publicKey"`
	PostUp              []string `json:"postUp"`
	PostDown            []string `json:"postDown"`
	PublicEndpoint      string   `json:"publicEndpoint"`
}

// Peer is a section of peer in a wg server config file
type Peer struct {
	Metadata `json:"metadata"`

	Spec   PeerSpec   `json:"spec"`
	Status PeerStatus `json:"status"`
}

// PeerSpec main configuration of a peer
type PeerSpec struct {
	PersistentPublicKey *Key                 `json:"persistentPublicKey"`
	AllowedIPs          string               `json:"allowedIPs"`
	ExpireAction        PeerExpireActionType `json:"expireAction"`
	ExpireDuration      string               `json:"expireDuration"`
	ClientMTU           string               `json:"clientMTU"`
	Blocked             bool                 `json:"blocked"`
}

// PeerStatus hold status of a peer
type PeerStatus struct {
	SecretValue string `json:"secretValue"`
	PublicKey   *Key   `json:"publicKey"`
}

// PeerClientConfig represents a Peer section on a client wireguard config
// https://git.zx2c4.com/WireGuard/about/src/tools/man/wg.8
type PeerClientConfig struct {
	PublicKey           string      `json:"publicKey"`
	AllowedIPs          []net.IPNet `json:"allowedIPs"`
	Endpoint            string      `json:"endpoint"`
	PersistentKeepAlive int         `json:"persistentKeepAlive"`
}
