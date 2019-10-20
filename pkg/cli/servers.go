package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

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
			fmt.Fprintln(w, "UID\tADDRESS\tPORT\tPUBKEY\tPEERS\t")
			for _, wg := range wgscList {
				pubkey := wg.PrivateKey.PublicKey().String()
				fmt.Fprintf(w, "%s\t%s\t%v\t%s\t%v\t", wg.UID, wg.Address, wg.ListenPort, pubkey, len(wg.ActivePeers))
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
			if wgsc != nil {
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
			err = client.WireguardServerConfig().Update(&api.WireguardServerConfig{
				Metadata: api.Metadata{
					UID:       wgenv,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				},
				Address:        api.ParseCIDR(O.Server.Address),
				PublicEndpoint: O.Server.PublicEndpoint,
				ListenPort:     O.Server.ListenPort,
				PrivateKey:     &privKey,
				PostUp: []string{
					"ip link set mtu 1500 dev ens4",
					"ip link set mtu 1500 dev %i",

					"sysctl -w net.ipv4.ip_forward=1",
					"sysctl -w net.ipv6.conf.all.forwarding=1",
					"iptables -A FORWARD -o %i -j ACCEPT",
					"iptables -t nat -A POSTROUTING -o ens4 -j MASQUERADE",
					"wg addconf %i /etc/wireguard/conf.d/peers.conf",
				},
				PostDown: []string{
					"sysctl -w net.ipv4.ip_forward=0",
					"sysctl -w net.ipv6.conf.all.forwarding=0",
					"iptables -D FORWARD -o %i -j ACCEPT",
					"iptables -t nat -D POSTROUTING -o ens4 -j MASQUERADE",
				},
			})
			if err != nil {
				return fmt.Errorf("failed creating wireguard server config: %v", err)
			}
			return client.SyncRemote()
		},
	}
	cmd.Flags().StringVar(&O.Server.Address, "address", "192.168.180.1/32", "The address of wireguard server config.")
	cmd.Flags().StringVar(&O.Server.PublicEndpoint, "endpoint", "", "The public [DNS|IP]:PORT for the wireguard server instance.")
	cmd.Flags().IntVar(&O.Server.ListenPort, "listen-port", 51820, "The listen port for the wireguard server.")
	return cmd
}
