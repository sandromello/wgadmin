package cli

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sandromello/wgadmin/pkg/store"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
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
	HTTPPort string
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

// func InitServer() error {
// 	fi, err := os.Stat(WGAppConfigPath)
// 	if os.IsNotExist(err) {
// 		if err := os.MkdirAll(WGAppConfigPath, 0744); err != nil {
// 			return err
// 		}
// 		return nil
// 	}
// 	if !fi.Mode().IsDir() {
// 		return fmt.Errorf("wgapp config path %q is a file", WGAppConfigPath)
// 	}
// 	return nil
// }

// func StorePreLoad(cmd *cobra.Command, args []string) {
// 	// expected <bucket>/<resource> or <bucket>
// 	parts := strings.Split(args[0], "/")
// 	dbfile := filepath.Join(WGAppConfigPath, "store.db")
// 	StoreClient = storeclient.NewOrDie(dbfile, parts[0], nil)
// }
