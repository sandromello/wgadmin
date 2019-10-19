package cli

import (
	"os"
	"path/filepath"

	"github.com/sandromello/wgadmin/pkg/store"
)

type CmdServer struct {
	Address        string
	ListenPort     int
	PublicEndpoint string
}

type CmdPeer struct {
	PublicAddressURL string
	Address          string
	Override         bool
}

type CmdOptions struct {
	ShowVersionAndExit bool
	JSONFormat         bool

	Server CmdServer
	Peer   CmdPeer
}

var (
	O CmdOptions

	WGAppConfigPath = os.ExpandEnv("$HOME/.wgapp")
	DBFile          = filepath.Join(WGAppConfigPath, store.DBFileName)
)

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
