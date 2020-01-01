package cli

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/sandromello/wgadmin/pkg/api"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/pkg/systemd"
	"github.com/sandromello/wgadmin/pkg/wgtools"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func hashFromByte(data []byte) string {
	s := sha1.New()
	s.Write(data)
	return string(s.Sum(nil))
}

// func fetchState(server, configFile, cipherKey string) ([]byte, []byte, error) {
func fetchState(sc *api.ServerConfig) ([]byte, []byte, error) {
	localConfigData, err := ioutil.ReadFile(sc.GetWireguardConfigFile())
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	// TODO: set a timeout when opening: bolt.Options{Timeout: Duration}
	client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()
	wgsc, err := client.WireguardServerConfig().Get(sc.Name)
	if err != nil {
		return nil, nil, err
	}
	if wgsc == nil {
		return nil, nil, fmt.Errorf("wireguard server %q not found", sc.Name)
	}
	remoteConfigData, err := wgsc.ParseWireguardServerConfigTemplate(sc.ServerDaemon.CipherKey)
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

func parseServerConfigFile(path string) (*api.ServerConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c api.ServerConfig
	return &c, yaml.Unmarshal(data, &c)
}

// InstallDaemons install and configure all daemons on the system
func InstallDaemons() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "install-daemons",
		Short:        "Install and configure systemd manager daemons.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			sc, err := parseServerConfigFile(O.ServerConfigPath)
			if err != nil {
				return err
			}
			fmt.Println("Installing", sc.ServerDaemon.GetUnitName())
			systemd.StopDaemon(&sc.ServerDaemon)
			if err := systemd.InstallWireguardManager(&sc.ServerDaemon); err != nil {
				return err
			}
			fmt.Println("Installing", sc.PeerDaemon.GetUnitName())
			systemd.StopDaemon(&sc.PeerDaemon)
			return systemd.InstallPeerManager(&sc.PeerDaemon)
		},
	}
	cmd.Flags().StringVarP(&O.ServerConfigPath, "config-file", "c", "", "The wgadmin config file.")
	return cmd
}

// SyncServerCmd syncronizes server configuration
func SyncServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "sync-servers",
		Short:             "Synchronize servers in Wireguard.",
		SilenceUsage:      true,
		PersistentPreRunE: PersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			sc, err := parseServerConfigFile(O.ServerConfigPath)
			if err != nil {
				return err
			}
			// Configuring runtime defaults
			os.Setenv("GCS_BUCKET_NAME", sc.BucketName)
			if sc.ServerDaemon.CipherKey == "" {
				sc.ServerDaemon.CipherKey = os.Getenv("CIPHER_KEY")
				if sc.ServerDaemon.CipherKey == "" {
					return fmt.Errorf("Cipher Key is not set")
				}
			}
			conciliate := func(logf *log.Entry) error {
				logf.Infof("Synchronize server %s ...", sc.Name)
				localData, remoteData, err := fetchState(sc)
				if err != nil {
					return err
				}
				wireguardConfigFile := sc.GetWireguardConfigFile()
				isDirty, err := checkDirty(wireguardConfigFile)
				if err != nil {
					return err
				}
				isConciliateOperation := hashFromByte(localData) != hashFromByte(remoteData)
				logf.Infof("dirty=%v, conciliate=%v", isDirty, isConciliateOperation)
				if isConciliateOperation || isDirty {
					stdout, err := conciliateState(wireguardConfigFile, remoteData)
					if err != nil {
						return fmt.Errorf("%v. %v", strings.TrimSuffix(string(stdout), "\n"), err)
					}
					logf.Debug(string(stdout))
				}
				return nil
			}

			isControlLoop := sc.ServerDaemon.SyncTime != api.Duration(0)
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
				time.Sleep(time.Duration(sc.ServerDaemon.SyncTime))
			}
		},
	}
	cmd.Flags().StringVarP(&O.ServerConfigPath, "config-file", "c", "", "The wgadmin config file.")
	return cmd
}

// SyncPeersCmd synchronize peers configuration
func SyncPeersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "sync-peers",
		Short:             "Synchronize peers in a Wireguard Server.",
		SilenceUsage:      true,
		PersistentPreRunE: PersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configuring runtime defaults
			sc, err := parseServerConfigFile(O.ServerConfigPath)
			if err != nil {
				return err
			}
			os.Setenv("GCS_BUCKET_NAME", sc.BucketName)
			iface := sc.PeerDaemon.InterfaceName
			conciliate := func(logf *log.Entry) error {
				client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
				if err != nil {
					return err
				}
				defer client.Close()
				// remove any revoked peers found in remote
				desiredPeers, err := client.Peer().ListByServer(sc.Name)
				if err != nil {
					return fmt.Errorf("failed listing peers: %v", err)
				}
				dirty := 0
				for _, peer := range desiredPeers {
					shouldAutoLock := peer.ShouldAutoLock()
					logf.Debugf("op=revoke, peer=%v, status=%v, autolock=%v", peer.UID, peer.GetStatus(), shouldAutoLock)
					if peer.GetStatus() == api.PeerBlocked || peer.GetStatus() == api.PeerActive && shouldAutoLock {
						logf.Infof("Removing dirty peer %v", peer.UID)
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
					cur, err := client.Peer().SearchByPubKey(sc.Name, pubkey)
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
					// don't process blocked, locked or expired peers
					if desired.GetStatus() != api.PeerActive || desired.ShouldAutoLock() {
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
						stdout, err := wgtools.WGAddPeer(iface, pubkey, desired.Spec.AllowedIPs)
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
			isControlLoop := sc.PeerDaemon.SyncTime != api.Duration(0)
			for {
				now := time.Now().UTC()
				logf := log.WithFields(map[string]interface{}{
					"job":    uuid.New().String()[:6],
					"server": sc.Name,
				})
				if err := conciliate(logf); err != nil {
					if !isControlLoop {
						return err
					}
					logf.Error(err)
				}
				logf.Infof("Completed in %vs", time.Since(now).Seconds())
				time.Sleep(time.Duration(sc.PeerDaemon.SyncTime))
			}
		},
	}
	cmd.Flags().StringVarP(&O.ServerConfigPath, "config-file", "c", "", "The wgadmin config file.")
	return cmd
}
