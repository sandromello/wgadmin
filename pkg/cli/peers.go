package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sandromello/wgadmin/pkg/api"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/pkg/util"
	"github.com/spf13/cobra"
)

// PeerAddCmd add a new peer
func PeerAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "add SERVER/PEER",
		Short:        "Add a new peer to a wireguard server config.",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing the resource name")
			}
			if !strings.Contains(args[0], "/") {
				return errors.New("specify the resource name as <SERVER>/<NAME>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			// <server>/<peer>
			parts := strings.Split(args[0], "/")
			p, err := client.Peer().Get(args[0])
			if err == nil && (p == nil || O.Peer.Override) {
				randomString, err := util.GenerateRandomString(50)
				if err != nil {
					return fmt.Errorf("failed generating random string: %v", err)
				}
				var allowedIPs *net.IPNet
				if O.Peer.Address != "" {
					allowedIPs = api.ParseCIDR(O.Peer.Address)
					if allowedIPs == nil {
						return fmt.Errorf("failed parsing ip address: %v", O.Peer.Address)
					}
				}

				wgsc, err := client.WireguardServerConfig().Get(parts[0])
				if err != nil || wgsc == nil {
					return fmt.Errorf("failed fetching server %v, err=%v", parts[0], err)
				}

				peerList, err := client.Peer().ListByServer(wgsc.UID)
				if err != nil {
					return fmt.Errorf("failed listing peers. err=%v", err)
				}
				ipmap, err := util.NewIPMap(wgsc.Address)
				if err != nil {
					return fmt.Errorf("failed creating ip map. err=%v", err)
				}
				for _, p := range peerList {
					ipmap.Del(p.AllowedIPs.IP)
				}
				if allowedIPs == nil {
					allowedIPs = ipmap.Pop()
					if allowedIPs == nil {
						return fmt.Errorf("reach maximum allocation for network %v, found %v peers", ipmap.Net.String(), len(peerList))
					}
				} else {
					if !ipmap.Net.Contains(allowedIPs.IP) {
						return fmt.Errorf("ip=%s doesn't belong to network=%v", allowedIPs.IP.String(), ipmap.Net.String())
					}
					if !ipmap.IsAvailable(allowedIPs.IP) {
						return fmt.Errorf("the ip=%v isn't available", allowedIPs.IP.String())
					}
				}
				randomSecret := fmt.Sprintf("%s.conf", randomString)
				if err := client.Peer().Update(&api.Peer{
					Metadata: api.Metadata{
						UID:       args[0],
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
					},
					PublicKey:      nil, // will be set when downloading the config
					ExpireAction:   wgsc.PeerExpireAction,
					ExpireDuration: O.Peer.ParseExpireDuration("24h"),
					AllowedIPs:     *allowedIPs,
					Status:         api.PeerStatusInitial,
					SecretValue:    randomSecret,
				}); err != nil {
					return err
				}
				wgenv := strings.Split(args[0], "/")[0]
				fmt.Printf("%s/peers/%s?vpn=%s\n", O.Peer.PublicAddressURL, randomSecret, wgenv)
			} else if err != nil {
				// failed veryfing if peer exists
				return fmt.Errorf("failed fetching peer: %v", err)
			} else if p != nil {
				return fmt.Errorf("peer already exists: %v", p.UID)
			}
			return client.SyncRemote()
		},
	}
	cmd.Flags().StringVar(&O.Peer.PublicAddressURL, "public-address", "http://127.0.0.1", "The public address that will be used to download the wireguard client config.")
	cmd.Flags().StringVar(&O.Peer.Address, "address", "", "The address of the peer, must not overlap with other peers.")
	cmd.Flags().StringVar(&O.Peer.ExpireDuration, "expire-in", "24h", "The duration for auto expiring or locking the peer.")
	cmd.Flags().BoolVar(&O.Peer.Override, "override", false, "Override the configured peer, it will reset the current configuration.")
	return cmd
}

func PeerInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "info PEER",
		Short:        "Get information about a specific peer.",
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
			peer, err := client.Peer().Get(args[0])
			if err != nil {
				return err
			}
			if peer == nil {
				return fmt.Errorf("peer not found")
			}
			if O.JSONFormat {
				jsonData, err := json.Marshal(peer)
				if err != nil {
					return fmt.Errorf("failed to serialize to json format: %v", err)
				}
				fmt.Println(string(jsonData))
				return nil
			}
			pubkey := "-"
			if peer.PublicKey != nil {
				pubkey = peer.PublicKey.String()
			}
			fmt.Println("UID:", peer.UID)
			fmt.Println("CREATEDAT:", peer.CreatedAt)
			fmt.Println("UPDATEDAT:", peer.UpdatedAt)
			fmt.Println("PUBKEY:", pubkey)
			fmt.Println("SECRET:", peer.SecretValue)
			fmt.Println("ALLOWEDIPS:", peer.AllowedIPs.String())
			fmt.Println("AUTOLOCK:", peer.ShouldAutoLock())
			fmt.Println("STATUS:", peer.GetStatus())
			return nil
		},
	}
	cmd.Flags().BoolVar(&O.JSONFormat, "json", false, "Print the output in JSON format.")
	return cmd
}

func PeerSetStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-status PEER [initial|active|blocked]",
		Short: "Set the status for a peer.",
		Long: string([]byte(`Set the status for a peer.
blocked: The user will not be able to establish connection with the server neither download the configuration.
initial: The user will need to download the configuration before the expiration time. Set to initial will reset the configuration of a client.
active: The user has dowloaded the client configuration and it's ready to establish connection with the server.
		`)),
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("missing required parameters: PEER [initial|active|blocked]")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			peerStatus := api.PeerStatus(args[1])
			switch peerStatus {
			case "initial":
				peerStatus = api.PeerStatusInitial
			case api.PeerStatusBlocked:
			case api.PeerStatusActive:
			default:
				return fmt.Errorf("wrong peer status %q", peerStatus)
			}
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			peer, err := client.Peer().Get(args[0])
			if err != nil {
				return err
			}
			if peer == nil {
				return fmt.Errorf("peer not found")
			}
			peer.Status = peerStatus
			if peer.Status == api.PeerStatusInitial {
				randomString, err := util.GenerateRandomString(50)
				if err != nil {
					return fmt.Errorf("failed generating random string: %v", err)
				}
				peer.PublicKey = nil
				peer.SecretValue = fmt.Sprintf("%s.conf", randomString)
			}
			if peer.PublicKey == nil && peer.Status == api.PeerStatusActive {
				return fmt.Errorf("cannot set to active without a public key")
			}
			if err := client.Peer().Update(peer); err != nil {
				return err
			}
			defer fmt.Printf("peer %q in %s state!\n", peer.UID, peer.GetStatus())
			return client.SyncRemote()
		},
	}
	cmd.Flags().BoolVar(&O.JSONFormat, "json", false, "Print the output in JSON format.")
	return cmd
}

func PeerListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list [SERVER]",
		Short:        "List peers from a given server.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			var serverPrefix string
			if len(args) > 0 {
				serverPrefix = args[0]
			}
			peerList, err := client.Peer().ListByServer(serverPrefix)
			if err != nil {
				return err
			}
			if O.JSONFormat {
				jsonList, err := json.Marshal(peerList)
				if err != nil {
					return fmt.Errorf("failed to serialize to json format: %v", err)
				}
				fmt.Println(string(jsonList))
				return nil
			}
			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 8, 2, '\t', tabwriter.AlignRight)
			defer w.Flush()
			if len(peerList) == 0 {
				fmt.Println("No resources found.")
				return nil
			}

			fmt.Fprintln(w, "UID\tALLOWEDIP\tSECRET\tPUBKEY\tSTATUS\tEXPIRE IN\tUPDATED AT\t")
			for _, p := range peerList {
				var pubkey string
				if p.PublicKey != nil {
					pubkey = p.PublicKey.String()
					prefixPubKey := pubkey[0:6]
					suffixPubKey := pubkey[len(pubkey)-6 : len(pubkey)]
					pubkey = fmt.Sprintf("%s...%s", prefixPubKey, suffixPubKey)
				}
				var secret string
				if len(p.SecretValue) > 0 {
					prefixSecret := p.SecretValue[0:5]
					suffixSecret := p.SecretValue[len(p.SecretValue)-10 : len(p.SecretValue)]
					secret = fmt.Sprintf("%s...%s", prefixSecret, suffixSecret)
				}
				ipaddr := p.AllowedIPs.String()
				var expin string
				switch d := p.GetExpirationDuration(); {
				case d <= 0 || p.PublicKey == nil:
					expin = "-"
				case d.Hours() > 24:
					expin = fmt.Sprintf("%.fd", math.Floor(d.Hours()/24))
				default:
					expin = d.String()
				}
				updatedAt := util.GetDeltaDuration(p.UpdatedAt, "")
				fmt.Fprintf(w, "%s\t%s\t%v\t%s\t%v\t%v\t%v\t", p.UID, ipaddr, secret, pubkey, p.GetStatus(), expin, updatedAt)
				fmt.Fprintln(w)
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&O.JSONFormat, "json", false, "Print the output in JSON format.")
	return cmd
}
