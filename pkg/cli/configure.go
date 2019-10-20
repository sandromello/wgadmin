package cli

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/spf13/cobra"
)

func hashFromByte(data []byte) string {
	s := sha1.New()
	s.Write(data)
	return string(s.Sum(nil))
}

func configureServer(server, configFile string) error {
	localConfigData, err := ioutil.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
	if err != nil {
		return err
	}
	wgsc, err := client.WireguardServerConfig().Get(server)
	if err != nil {
		return err
	}
	if wgsc == nil {
		return fmt.Errorf("wireguard server %q not found", server)
	}
	remoteConfigData, err := wgsc.ParseWireguardServerConfigTemplate()
	if err != nil {
		return err
	}
	if hashFromByte(localConfigData) != hashFromByte(remoteConfigData) {
		fmt.Println("local state has changed, reconfiguring server ...")
		// TODO: overwrite the destination configuration with the remote config
		// TODO: wgtools.wgQuickDown() - to turn off the interface
		// TODO: wgtools.wgQuickUP() - to turn on the interface
	}
	fmt.Println("LOCAL == REMOTE:", hashFromByte(localConfigData) == hashFromByte(remoteConfigData))
	return client.SyncRemote()
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
			for {
				errCh := make(chan error, 1)
				go func() {
					errCh <- configureServer(args[0], O.Configure.ConfigFile)
				}()
				select {
				case err := <-errCh:
					isControlLoop := O.Configure.Sync != time.Duration(0)
					if err != nil {
						if !isControlLoop {
							return err
						}
						fmt.Println(err.Error())
					}
					if !isControlLoop {
						return nil
					}
					time.Sleep(O.Configure.Sync)
				case <-time.After(5 * time.Second):
					return fmt.Errorf("timeout")
				}
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
