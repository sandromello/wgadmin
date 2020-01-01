package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sandromello/wgadmin/pkg/api"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/pkg/util"
	"github.com/spf13/cobra"
)

// PeerApplyCmd creates resources using yaml files
func PeerApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "apply",
		Short:        "Add a peers based on a declared yaml file.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := ioutil.ReadFile(O.Peer.Filename)
			if err != nil {
				return err
			}
			peerList := []api.Peer{}
			if err := yaml.Unmarshal(file, &peerList); err != nil {
				return err
			}
			sort.Sort(api.SortPeerByUID(peerList))
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			success := 0
			for _, new := range peerList {
				peerServer := new.GetServer()
				// TODO: There's no need to fetch a server for every single peer config, improve later
				wgsc, err := client.WireguardServerConfig().Get(peerServer)
				if err != nil || wgsc == nil {
					return fmt.Errorf("failed fetching server %v, err=%v", peerServer, err)
				}
				old, err := client.Peer().Get(new.UID)
				if err != nil {
					return fmt.Errorf("failed fetching peer %s, err=%v", new.UID, err)
				}
				ipmap, err := buildIPMap(client, wgsc)
				if err != nil {
					return err
				}
				if old == nil {
					if err := validatePeer(&new); err != nil {
						return fmt.Errorf("failed validating peer %s, err=%v", new.UID, err)
					}
					ipaddr := new.ParseAllowedIPs()
					if !ipmap.IsAvailable(ipaddr) {
						return fmt.Errorf("peer %s has ip %q which isn't available for network %v", new.UID, ipaddr.String(), ipmap.Net.String())
					}
					new.CreatedAt = time.Now().UTC().Format(time.RFC3339)
					if err := client.Peer().Update(&new); err != nil {
						return fmt.Errorf("failed creating new peer %s, err=%v", new.UID, err)
					}
					success++
				} else if !reflect.DeepEqual(old.Spec, new.Spec) {
					ipaddr := new.ParseAllowedIPs()
					if old.Spec.AllowedIPs != new.Spec.AllowedIPs && !ipmap.IsAvailable(ipaddr) {
						return fmt.Errorf("peer %s has ip %q which isn't available for network %v", new.UID, ipaddr.String(), ipmap.Net.String())
					}
					new.Metadata = old.Metadata
					new.Status = old.Status
					if err := client.Peer().Update(&new); err != nil {
						return fmt.Errorf("failed updating peer %s, err=%v", new.UID, err)
					}
					success++
				}
			}
			if err := client.SyncRemote(); err != nil {
				return err
			}
			if success == 0 {
				fmt.Printf("Nothing changed.\n")
			} else {
				fmt.Printf("%d/%d peer(s) updated.\n", success, len(peerList))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&O.Peer.Filename, "filename", "f", "", "Contains the configuration to apply.")
	return cmd
}

func validatePeer(p *api.Peer) error {
	switch p.Spec.ExpireAction {
	case api.PeerExpireActionBlock:
	case api.PeerExpireActionReset:
	case api.PeerExpireActionDefault:
	default:
		return errors.New("not a valid expireAction option")
	}
	var err error
	if p.Spec.ExpireDuration != "" {
		_, err = time.ParseDuration(p.Spec.ExpireDuration)
	}
	return err
}

func buildIPMap(client storeclient.Client, wgsc *api.WireguardServerConfig) (*util.IPMap, error) {
	peerList, err := client.Peer().ListByServer(wgsc.UID)
	if err != nil {
		return nil, fmt.Errorf("failed listing peers. err=%v", err)
	}
	ipmap, err := util.NewIPMap(wgsc.Address)
	if err != nil {
		return nil, fmt.Errorf("failed creating ip map. err=%v", err)
	}
	for _, p := range peerList {
		ipmap.Del(p.ParseAllowedIPs().String())
	}
	return ipmap, nil
}

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
				persistentPubKey, err := O.ParsePersistentPublicKey()
				if err != nil {
					return fmt.Errorf("failed parsing persistent public key: %v", err)
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
				ipmap, err := buildIPMap(client, wgsc)
				if err != nil {
					return err
				}
				if allowedIPs == nil {
					allowedIPs = ipmap.Pop()
					if allowedIPs == nil {
						return fmt.Errorf("reach maximum allocation for network %v", ipmap.Net.String())
					}
				} else {
					if !ipmap.Net.Contains(allowedIPs.IP) {
						return fmt.Errorf("ip=%s doesn't belong to network=%v", allowedIPs.IP.String(), ipmap.Net.String())
					}
					if !ipmap.IsAvailable(allowedIPs.IP) {
						return fmt.Errorf("the ip=%v isn't available", allowedIPs.IP.String())
					}
				}
				var wireguardClientConfig []byte
				if O.Peer.ClientConfig {
					clientPrivkey, err := api.GeneratePrivateKey()
					if err != nil {
						return fmt.Errorf("failed generating private key for client config, err=%v", err)
					}
					pubkey := clientPrivkey.PublicKey()
					persistentPubKey = &pubkey
					wireguardClientConfig, err = api.ParseWireguardClientConfigTemplate(map[string]interface{}{
						"PrivateKey": clientPrivkey,
						"PublicKey":  wgsc.PublicKey.String(),
						"Address":    allowedIPs.String(),
						"DNS":        "1.1.1.1, 8.8.8.8",
						"Endpoint":   wgsc.PublicEndpoint,
						"AllowedIPs": "0.0.0.0/0, ::/0",
					})
					if err != nil {
						return fmt.Errorf("failed generating client config, err=%v", err)
					}
				}
				if err := client.Peer().Update(&api.Peer{
					Metadata: api.Metadata{
						UID:       args[0],
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
					},
					Spec: api.PeerSpec{
						PersistentPublicKey: persistentPubKey,
						// TODO: validate expire action first
						ExpireAction: api.PeerExpireActionType(O.Peer.ExpireAction),
						// TODO: parse expire duration
						ExpireDuration: "24h",
						AllowedIPs:     allowedIPs.String(),
					},
				}); err != nil {
					return err
				}
				if wireguardClientConfig != nil {
					fmt.Print(string(wireguardClientConfig))
				}
			} else if err != nil {
				// failed veryfing if peer exists
				return fmt.Errorf("failed fetching peer: %v", err)
			} else if p != nil {
				return fmt.Errorf("peer already exists: %v", p.UID)
			}
			return client.SyncRemote()
		},
	}
	cmd.Flags().StringVar(&O.Peer.Address, "address", "", "The address of the peer, must not overlap with other peers.")
	cmd.Flags().StringVar(&O.Peer.ExpireAction, "expire-action", string(api.PeerExpireActionDefault), "The action to perform when expiring peers: block|reset.")
	cmd.Flags().StringVar(&O.Peer.ExpireDuration, "expire-in", "24h", "The duration for auto expiring or locking the peer.")
	cmd.Flags().StringVar(&O.Peer.PersistentPublicKey, "public-key", "", "The public key to add to the peer, this key will never expire.")
	cmd.Flags().BoolVar(&O.Peer.ClientConfig, "client-config", false, "Generate a wireguard client config, this public key will never expire.")
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
			if O.Output != "" {
				return O.PrintOutputOptionToStdout(peer)
			}
			pubkey := "-"
			if peer.GetPublicKey() != nil {
				pubkey = peer.GetPublicKey().String()
			}
			expireAction := "-"
			if peer.Spec.ExpireAction != "" {
				expireAction = string(peer.Spec.ExpireAction)
			}
			expireDuration := "-"
			if peer.Spec.ExpireDuration != "" {
				expireDuration = peer.Spec.ExpireDuration
			}
			fmt.Println("UID:", peer.UID)
			fmt.Println("CREATEDAT:", peer.CreatedAt)
			fmt.Println("UPDATEDAT:", peer.UpdatedAt)
			fmt.Println("PUBKEY:", pubkey)
			fmt.Println("SECRET:", peer.Status.SecretValue)
			fmt.Println("BLOCKED:", peer.Spec.Blocked)
			fmt.Println("EXPIREACTION:", expireAction)
			fmt.Println("EXPIREDURATION:", expireDuration)
			fmt.Println("ALLOWEDIPS:", peer.Spec.AllowedIPs)
			fmt.Println("AUTOLOCK:", peer.ShouldAutoLock())
			fmt.Println("STATUS:", peer.GetStatus())
			return nil
		},
	}
	cmd.Flags().StringVarP(&O.Output, "output", "o", "", "Output format. One of: json|yaml.")
	return cmd
}

// PeerBlockCmd block a given peer
func PeerBlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "block PEER",
		Short:        "Block a given peer",
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
			peer, err := client.Peer().Get(args[0])
			if err != nil {
				return err
			}
			if peer == nil {
				return fmt.Errorf("peer not found")
			}
			peer.Spec.Blocked = true
			if err := client.Peer().Update(peer); err != nil {
				return err
			}
			fmt.Printf("peer %q is blocked!\n", peer.UID)
			return client.SyncRemote()
		},
	}
}

// PeerUnblockCmd unblock a given peer
func PeerUnblockCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "unblock PEER",
		Short:        "Unblock a given peer",
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
			peer, err := client.Peer().Get(args[0])
			if err != nil {
				return err
			}
			if peer == nil {
				return fmt.Errorf("peer not found")
			}
			peer.Spec.Blocked = false
			if err := client.Peer().Update(peer); err != nil {
				return err
			}
			fmt.Printf("peer %q is active!\n", peer.UID)
			return client.SyncRemote()
		},
	}
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
			if O.Output != "" {
				return O.PrintOutputOptionToStdout(peerList)
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
				if p.Spec.PersistentPublicKey != nil {
					p.UID = fmt.Sprintf("%s*", p.UID) // Indicates that is a persistent pub key
				}
				if p.GetPublicKey() != nil {
					pubkey = p.GetPublicKey().String()
					prefixPubKey := pubkey[0:6]
					suffixPubKey := pubkey[len(pubkey)-6 : len(pubkey)]
					pubkey = fmt.Sprintf("%s...%s", prefixPubKey, suffixPubKey)

				}
				var secret string
				if len(p.Status.SecretValue) > 0 {
					prefixSecret := p.Status.SecretValue[0:5]
					suffixSecret := p.Status.SecretValue[len(p.Status.SecretValue)-10 : len(p.Status.SecretValue)]
					secret = fmt.Sprintf("%s...%s", prefixSecret, suffixSecret)
				}
				ipaddr := p.Spec.AllowedIPs
				var expin string
				switch d := p.GetExpirationDuration(); {
				case p.Spec.ExpireAction == api.PeerExpireActionDefault || p.Spec.PersistentPublicKey != nil:
					expin = "never"
				case d <= 0 || p.GetPublicKey() == nil:
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
	cmd.Flags().StringVarP(&O.Output, "output", "o", "", "Output format. One of: json|yaml.")
	return cmd
}
