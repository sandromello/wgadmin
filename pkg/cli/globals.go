package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/store"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/pkg/webapp"
	bolt "go.etcd.io/bbolt"
)

type CmdServer struct {
	Address        string
	ListenPort     int
	PublicEndpoint string
	InterfaceName  string
	Override       bool
	CipherKey      string
}

type CmdPeer struct {
	Address             string
	ExpireAction        string
	ExpireDuration      string
	PersistentPublicKey string
	ClientConfig        bool
	Override            bool
	Filename            string
}

type CmdConfigure struct {
	ConfigFile    string
	InterfaceName string
	CipherKey     string
	Sync          time.Duration
}

type CmdWebServer struct {
	HTTPPort       string
	AllowedDomains *[]string
	PageConfig     webapp.PageConfig
	TLSKeyFile     string
	TLSCertFile    string
}

type CmdOptions struct {
	ShowVersionAndExit bool
	ServerConfigPath   string
	Output             string
	Local              bool

	Server    CmdServer
	Peer      CmdPeer
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

// ParsePersistentPublicKey parse a base64 string pubkey to an api.Key
func (o *CmdOptions) ParsePersistentPublicKey() (*api.Key, error) {
	if o.Peer.PersistentPublicKey == "" {
		return nil, nil
	}
	k, err := api.ParseKey(o.Peer.PersistentPublicKey)
	return &k, err
}

// PrintOutputOptionToStdout print the object to the given format specified
func (o *CmdOptions) PrintOutputOptionToStdout(obj interface{}) error {
	switch o.Output {
	case "json":
		jsonList, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("Error: failed to serialize to json format: %v", err)
		}
		fmt.Println(string(jsonList))
	case "yaml":
		yamlList, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("Error: failed to serialize to yaml format: %v", err)
		}
		fmt.Print(string(yamlList))
	default:
		return errors.New("wrong output option specified")
	}
	return nil
}

// InitEmptyBoltOptions initialize an empty bolt.Options
func InitEmptyBoltOptions() *bolt.Options {
	return &bolt.Options{}
}
