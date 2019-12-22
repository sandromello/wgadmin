package cli

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sandromello/wgadmin/pkg/store"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/web"
	bolt "go.etcd.io/bbolt"
)

type CmdServer struct {
	Address              string
	ListenPort           int
	PublicEndpoint       string
	PeerExpireActionType string
	InterfaceName        string
	Override             bool
}

type CmdPeer struct {
	PublicAddressURL string
	Address          string
	ExpireDuration   string
	Override         bool
}

func (c *CmdPeer) ParseExpireDuration(defaultDuration string) time.Duration {
	d, err := time.ParseDuration(c.ExpireDuration)
	if err != nil {
		d, _ = time.ParseDuration(defaultDuration)
	}
	return d
}

type CmdConfigure struct {
	ConfigFile    string
	InterfaceName string
	Sync          time.Duration
}

type CmdWebServer struct {
	HTTPPort       string
	AllowedDomains *[]string
	PageConfig     web.PageConfig
}

type CmdOptions struct {
	ShowVersionAndExit bool
	JSONFormat         bool
	Local              bool

	Server    CmdServer
	Peer      CmdPeer
	Configure CmdConfigure
	WebServer CmdWebServer
}

const (
	cmdTimeoutInSeconds = 2
)

var (
	O CmdOptions

	GlobalWGAppConfigPath = os.ExpandEnv("$HOME/.wgapp")
	GlobalDBFile          = filepath.Join(GlobalWGAppConfigPath, store.DBFileName)
	GlobalBoltOptions     = &bolt.Options{OpenFile: storeclient.FetchFromGCS}
)

// InitEmptyBoltOptions initialize an empty bolt.Options
func InitEmptyBoltOptions() *bolt.Options {
	return &bolt.Options{}
}
