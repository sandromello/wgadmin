package cli

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/sandromello/wgadmin/pkg/api"

	"github.com/google/uuid"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/pkg/system/wgpeer"
	"github.com/sandromello/wgadmin/pkg/system/wgserver"
	"github.com/sandromello/wgadmin/pkg/wgtools"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func hashFromByte(data []byte) string {
	s := sha1.New()
	s.Write(data)
	return string(s.Sum(nil))
}

func fetchState(server, configFile string) ([]byte, []byte, error) {
	localConfigData, err := ioutil.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	// TODO: set a timeout when opening: bolt.Options{Timeout: Duration}
	client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()
	wgsc, err := client.WireguardServerConfig().Get(server)
	if err != nil {
		return nil, nil, err
	}
	if wgsc == nil {
		return nil, nil, fmt.Errorf("wireguard server %q not found", server)
	}
	remoteConfigData, err := wgsc.ParseWireguardServerConfigTemplate()
	return localConfigData, remoteConfigData, err
}

func checkDirty(configFile string) (isDirty bool, errMsg error) {
	dirtyFile := fmt.Sprintf("%s.dirty", configFile)
	isDirty = true
	if _, err := os.Stat(dirtyFile); err != nil {
		if os.IsNotExist(err) {
			isDirty = false
		} else {
			errMsg = fmt.Errorf("failed reading dirty file state: %v", err)
		}
	}
	return
}

func setDirty(isDirty bool, configFile string) (errMsg error) {
	dirtyFile := fmt.Sprintf("%s.dirty", configFile)
	if isDirty {
		if err := ioutil.WriteFile(dirtyFile, []byte(``), 0744); err != nil {
			errMsg = fmt.Errorf("failed creating the dirty file state: %v", err)
		}
		return errMsg
	}
	if err := os.Remove(dirtyFile); err != nil {
		errMsg = fmt.Errorf("failed removing the dirty file state: %v", err)
	}
	return errMsg
}

func conciliateState(configFile string, data []byte) ([]byte, error) {
	if err := setDirty(true, configFile); err != nil {
		return nil, err
	}
	isRoot, err := wgtools.IsRootUser()
	if err != nil {
		return nil, fmt.Errorf("failed veryfing the current user: %v", err)
	}
	if !isRoot {
		return nil, fmt.Errorf("must be run as root user")
	}
	// TODO: overwrite the destination configuration with the remote config
	if err := ioutil.WriteFile(configFile, data, 0700); err != nil {
		return nil, err
	}
	// TODO: add vagrant for testing locally
	wgtools.WGQuickDown(configFile)
	stdout, err := wgtools.WGQuickUP(configFile)
	if err != nil {
		return stdout, err
	}
	return stdout, setDirty(false, configFile)
}

// ConfigureSystemdServerCmd configure the systemd for running
// the wireguard server manager
func ConfigureSystemdServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "systemd-server",
		Short:        "Configure Systemd for the Wireguard Server Manager.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			wgserver.StopWireguardManager()
			return wgserver.InstallWireguardManager()
		},
	}
	return cmd
}

// ConfigureSystemdPeerCmd configure the systemd for running
// the wireguard peer manager
func ConfigureSystemdPeerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "systemd-peer",
		Short:        "Configure Systemd for the Wireguard Peer Manager.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			wgpeer.StopPeerManager()
			return wgpeer.InstallPeerManager()
		},
	}
	return cmd
}

// ConfigureServerCmd configure a wireguard server command
func ConfigureServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "server SERVER",
		Short:        "Configure a Wireguard Server.",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing the resource name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			conciliate := func(logf *log.Entry) error {
				logf.Infof("Configuring server %s ...", args[0])
				localData, remoteData, err := fetchState(args[0], O.Configure.ConfigFile)
				if err != nil {
					return err
				}
				isDirty, err := checkDirty(O.Configure.ConfigFile)
				if err != nil {
					return err
				}
				isConciliateOperation := hashFromByte(localData) != hashFromByte(remoteData)
				logf.Infof("dirty=%v, conciliate=%v", isDirty, isConciliateOperation)
				if isConciliateOperation || isDirty {
					stdout, err := conciliateState(O.Configure.ConfigFile, remoteData)
					if err != nil {
						return fmt.Errorf("%v. %v", strings.TrimSuffix(string(stdout), "\n"), err)
					}
					logf.Debug(string(stdout))
				}
				return nil
			}

			isControlLoop := O.Configure.Sync != time.Duration(0)
			for {
				now := time.Now().UTC()
				logf := log.WithField("job", uuid.New().String()[:6])
				if err := conciliate(logf); err != nil {
					if !isControlLoop {
						return err
					}
					logf.Error(err)
				}
				logf.Infof("Completed in %vs\n", time.Since(now).Seconds())
				time.Sleep(O.Configure.Sync)
			}
		},
	}
	cmd.Flags().StringVar(&O.Configure.ConfigFile, "config", "/etc/wireguard/wg0.conf", "The wireguard server config file path.")
	cmd.Flags().DurationVar(&O.Configure.Sync, "sync", time.Duration(0), "If enable will run a control loop watching the changes from remote.")
	return cmd
}

// ConfigurePeersCmd configure peers command
func ConfigurePeersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "peer SERVER",
		Short:        "Configure the peers in a Wireguard Server.",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing the resource name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			iface := O.Configure.InterfaceName
			conciliate := func(logf *log.Entry) error {
				client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
				if err != nil {
					return err
				}
				defer client.Close()
				// remove any revoked peers found in remote
				desiredPeers, err := client.Peer().ListByServer(args[0])
				if err != nil {
					return fmt.Errorf("failed listing peers: %v", err)
				}
				dirty := 0
				for _, peer := range desiredPeers {
					logf.Debugf("op=revoke, peer=%v, status=%v", peer.UID, peer.GetStatus())
					if peer.Status == api.PeerStatusBlocked {
						logf.Infof("Removing revoked peer %v", peer.UID)
						stdout, err := wgtools.WGRemovePeer(iface, peer.PublicKeyString())
						if err != nil {
							msg := fmt.Sprintf("%v. %v", strings.TrimSuffix(string(stdout), "\n"), err)
							logf.Error(msg)
							dirty++
						}
					}
				}
				// check if the current peers is consistent with the remote state
				currentPeers, err := wgtools.WGShowPeers(iface)
				if err != nil {
					return fmt.Errorf("failed listing wireguard peers: %v", err)
				}
				for _, pubkey := range currentPeers {
					logf.Debugf("op=conciliate, peer=%s", pubkey)
					cur, err := client.Peer().SearchByPubKey(args[0], pubkey)
					if err != nil {
						return fmt.Errorf("failed retrieving peer from store: %v", err)
					}
					if cur == nil {
						logf.Infof("Removing local peer %s", pubkey)
						stdout, err := wgtools.WGRemovePeer(iface, pubkey)
						if err != nil {
							msg := fmt.Sprintf("%v. %v", strings.TrimSuffix(string(stdout), "\n"), err)
							logf.Error(msg)
							dirty++
						}
					}
				}

				currentPeers, err = wgtools.WGShowPeers(iface)
				if err != nil {
					return fmt.Errorf("failed listing wireguard peers: %v", err)
				}
				// add peers if doesn't exists locally
				for _, desired := range desiredPeers {
					if desired.Status != api.PeerStatusActive {
						continue
					}
					logf.Debugf("op=add, peer=%s, status=%v", desired.UID, desired.GetStatus())
					exists := false
					for _, curPubKey := range currentPeers {
						if curPubKey == desired.PublicKeyString() {
							exists = true
							break
						}
					}
					if !exists {
						pubkey := desired.PublicKeyString()
						logf.Debugf("Adding peer %v/%v", desired.UID, pubkey)
						stdout, err := wgtools.WGAddPeer(iface, pubkey, desired.AllowedIPs.String())
						if err != nil {
							msg := fmt.Sprintf("%v. %v", strings.TrimSuffix(string(stdout), "\n"), err)
							logf.Error(msg)
							dirty++
						}
					}
				}
				logf.WithField("dirty", dirty).Infof("Found %v local and %v remote peers", len(currentPeers), len(desiredPeers))
				return nil
			}
			isControlLoop := O.Configure.Sync != time.Duration(0)
			for {
				now := time.Now().UTC()
				logf := log.WithFields(map[string]interface{}{
					"job":    uuid.New().String()[:6],
					"server": args[0],
				})
				if err := conciliate(logf); err != nil {
					if !isControlLoop {
						return err
					}
					logf.Error(err)
				}
				logf.Infof("Completed in %vs", time.Since(now).Seconds())
				time.Sleep(O.Configure.Sync)
			}
		},
	}
	cmd.Flags().StringVar(&O.Configure.InterfaceName, "iface", "wg0", "The wireguard peers config file path.")
	cmd.Flags().DurationVar(&O.Configure.Sync, "sync", time.Duration(0), "If enable will run a control loop watching the changes from remote.")
	return cmd
}
