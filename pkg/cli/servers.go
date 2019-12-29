package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sandromello/wgadmin/pkg/util"

	"github.com/sandromello/wgadmin/pkg/api"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/spf13/cobra"
)

// ListServer list wireguard server config
func ListServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List wireguard servers configs.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			wgscList, err := client.WireguardServerConfig().List()
			if err != nil {
				return err
			}
			if O.JSONFormat {
				jsonList, err := json.Marshal(wgscList)
				if err != nil {
					return fmt.Errorf("Error: failed to serialize to json format: %v", err)
				}
				fmt.Println(string(jsonList))
				return nil
			}
			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 8, 2, '\t', tabwriter.AlignRight)
			defer w.Flush()
			if len(wgscList) == 0 {
				fmt.Println("No resources found.")
				return nil
			}
			fmt.Fprintln(w, "UID\tADDRESS\tPORT\tPUBKEY\tEXPIREACTION\tPEERS\t")
			for _, wg := range wgscList {
				pubkey := wg.PublicKey.String()
				fmt.Fprintf(w, "%s\t%s\t%v\t%s\t%v\t%v\t", wg.UID, wg.Address, wg.ListenPort, pubkey, wg.PeerExpireAction, len(wg.ActivePeers))
				fmt.Fprintln(w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&O.JSONFormat, "json", false, "Print the output in json format.")
	return cmd
}

// DeleteServer removes a wireguard server config
func DeleteServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete NAME",
		Short:        "Delete a wireguard server config.",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing the resource name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			if err := client.WireguardServerConfig().Delete(args[0]); err != nil {
				return err
			}
			if err := client.SyncRemote(); err != nil {
				return err
			}
			fmt.Printf("wireguard server %q removed!\n", args[0])
			return nil
		},
	}
	return cmd
}

// NewCipherKey creates a new cipher key in advance to use as parameter
// when initializing the server
func NewCipherKey() *cobra.Command {
	return &cobra.Command{
		Use:          "new-cipher-key",
		Short:        "Generate a random secure cipher key in advance to use as parameter when initializing the server.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cipherKey, err := util.NewAESCipherKey("")
			if err != nil {
				return err
			}
			fmt.Println(cipherKey.String())
			return nil
		},
	}
}

// InitServer initialize a new wireguard server if doesn't exists
func InitServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "init NAME",
		Short:        "Initialize the wireguard server creating a wireguard server config.",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing the resource name")
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			fi, err := os.Stat(GlobalWGAppConfigPath)
			if os.IsNotExist(err) {
				if err := os.MkdirAll(GlobalWGAppConfigPath, 0744); err != nil {
					return err
				}
				return nil
			}
			if !fi.Mode().IsDir() {
				return fmt.Errorf("wgapp config path %q is a file", GlobalWGAppConfigPath)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			wgenv := args[0]
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			wgsc, err := client.WireguardServerConfig().Get(wgenv)
			if err != nil {
				return err
			}
			if wgsc != nil && !O.Server.Override {
				return fmt.Errorf("wireguard server config %q already exists", wgsc.UID)
			}
			addr := api.ParseCIDR(O.Server.Address)
			if addr == nil {
				return fmt.Errorf("ip address %q in wrong format", O.Server.Address)
			}
			if !strings.Contains(O.Server.PublicEndpoint, ":") {
				return fmt.Errorf("public endpoint %q invalid format", O.Server.PublicEndpoint)
			}
			privKey, err := api.GeneratePrivateKey()
			if err != nil {
				return fmt.Errorf("failed generating private key: %v", err)
			}
			cipherKey, err := util.NewAESCipherKey(O.Server.CipherKey)
			if err != nil {
				return fmt.Errorf("failed generating AES encryption key: %v", err)
			}
			encPrivKey, err := cipherKey.EncryptMessage(privKey.String())
			if err != nil {
				return fmt.Errorf("failed encrypting private key: %v", err)
			}
			pubKey := privKey.PublicKey()
			expireAction := O.Server.PeerExpireActionType
			switch api.PeerExpireActionType(expireAction) {
			case api.PeerExpireActionBlock, api.PeerExpireActionReset:
			default:
				expireAction = ""
			}
			if err := client.WireguardServerConfig().Update(&api.WireguardServerConfig{
				Metadata: api.Metadata{
					UID:       wgenv,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				},
				Address:          O.Server.Address,
				PublicEndpoint:   O.Server.PublicEndpoint,
				PeerExpireAction: api.PeerExpireActionType(expireAction),
				ListenPort:       O.Server.ListenPort,
				// PrivateKey:       &privKey,
				EncryptedPrivateKey: encPrivKey,
				PublicKey:           &pubKey,
				PostUp: []string{
					// https://github.com/StreisandEffect/streisand/issues/1089#issuecomment-350400689
					fmt.Sprintf("ip link set mtu 1360 dev %s", O.Server.InterfaceName),
					"ip link set mtu 1360 dev %i",

					"sysctl -w net.ipv4.ip_forward=1",
					"sysctl -w net.ipv6.conf.all.forwarding=1",
					"iptables -A FORWARD -o %i -j ACCEPT",
					"iptables -A FORWARD -i %i -j ACCEPT",
					fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", O.Server.InterfaceName),
				},
				PostDown: []string{
					"sysctl -w net.ipv4.ip_forward=0",
					"sysctl -w net.ipv6.conf.all.forwarding=0",
					"iptables -D FORWARD -o %i -j ACCEPT",
					"iptables -D FORWARD -i %i -j ACCEPT",
					fmt.Sprintf("iptables -t nat -D POSTROUTING -o %s -j MASQUERADE", O.Server.InterfaceName),
				},
			}); err != nil {
				return fmt.Errorf("failed creating wireguard server config: %v", err)
			}
			if err := client.SyncRemote(); err != nil {
				return fmt.Errorf("failed syncing remote state: %v", err)
			}
			// THe Cipher Key was randomly generated, print to stdout
			if O.Server.CipherKey == "" {
				fmt.Println(cipherKey.String())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&O.Server.InterfaceName, "iface", "eth0", "The name of the interface which will be used to run scripts.")
	cmd.Flags().StringVar(&O.Server.Address, "address", "192.168.180.1/22", "The address of wireguard server config.")
	cmd.Flags().StringVar(&O.Server.PublicEndpoint, "endpoint", "", "The public [DNS|IP]:PORT for the wireguard server instance.")
	cmd.Flags().StringVar(&O.Server.PeerExpireActionType, "expire-action", "", "The action when the peer is expired: reset or block, default to not expire.")
	cmd.Flags().StringVar(&O.Server.CipherKey, "cipher-key", os.Getenv("CIPHER_KEY"), "A base64 encoded key used to encrypt the private key, could be set using CIPHER_KEY environment variable.")
	cmd.Flags().BoolVar(&O.Server.Override, "override", false, "Override the current configuration.")
	cmd.Flags().IntVar(&O.Server.ListenPort, "listen-port", 51820, "The listen port for the wireguard server.")
	return cmd
}
