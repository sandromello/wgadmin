package cli

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
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
	wgtools.WGQuickDown(configFile)
	stdout, err := wgtools.WGQuickUP(configFile)
	if err != nil {
		return stdout, err
	}
	return stdout, setDirty(false, configFile)
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
			isControlLoop := O.Configure.Sync != time.Duration(0)
			for {
				now := time.Now().UTC()
				logf := log.WithField("job", uuid.New().String()[:6])
				errCh := make(chan error, 1)
				stopC := make(chan struct{})

				go func() {
					logf.Infof("Configuring server %s ...", args[0])
					defer close(stopC)

					localData, remoteData, err := fetchState(args[0], O.Configure.ConfigFile)
					if err != nil {
						errCh <- err
					}
					isDirty, err := checkDirty(O.Configure.ConfigFile)
					if err != nil {
						errCh <- err
					}
					isConciliateOperation := hashFromByte(localData) != hashFromByte(remoteData)
					logf.Debugf("dirty=%v, conciliate=%v", isDirty, isConciliateOperation)
					if isConciliateOperation || isDirty {
						stdout, err := conciliateState(O.Configure.ConfigFile, remoteData)
						if err != nil {
							errCh <- err
						}
						logf.Debug(string(stdout))
					}
				}()

				select {
				case <-stopC:
					logf.Infof("Done in %vs\n", time.Since(now).Seconds())
				case err := <-errCh:
					if err != nil {
						if !isControlLoop {
							return err
						}
						logf.Error(err.Error())
					}
				}
				if !isControlLoop {
					return nil
				}
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
		Use:          "peer SERVER/PEER",
		Short:        "Configure the peers in a Wireguard Server.",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("missing the resource name")
			}
			if !strings.Contains(args[0], "/") {
				return errors.New("specify the resource name as <SERVER>/<PEER>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
			if err != nil {
				return err
			}
			return client.SyncRemote()
		},
	}
	cmd.Flags().StringVar(&O.Configure.ConfigFile, "config", "/etc/wireguard/conf.d/peers.conf", "The wireguard peers config file path.")
	cmd.Flags().DurationVar(&O.Configure.Sync, "sync", time.Duration(0), "If enable will run a control loop watching the changes from remote.")
	return cmd
}
