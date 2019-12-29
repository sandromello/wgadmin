package main

import (
	"fmt"
	"os"

	"github.com/sandromello/wgadmin/pkg/cli"
	"github.com/sandromello/wgadmin/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	// log.SetFormatter(&log.JSONFormatter{})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func main() {
	root := &cobra.Command{
		Use:   "wgapp",
		Short: "wgapp manages users and wireguard servers.",
		Run: func(cmd *cobra.Command, args []string) {
			if cli.O.ShowVersionAndExit {
				fmt.Println(string(version.JSON()))
				os.Exit(0)
			}
			cmd.Help()
		},
	}
	servers := &cobra.Command{
		Use:               "server",
		Aliases:           []string{"servers"},
		Short:             "Interact with wireguard server config resources.",
		PersistentPreRunE: cli.PersistentPreRunE,
		SilenceUsage:      true,
	}
	peers := &cobra.Command{
		Use:               "peer",
		Aliases:           []string{"peers"},
		Short:             "Interact with peer resources.",
		PersistentPreRunE: cli.PersistentPreRunE,
		SilenceUsage:      true,
	}
	configure := &cobra.Command{
		Use:               "configure",
		Short:             "Manage peers and server configurations.",
		PersistentPreRunE: cli.PersistentPreRunE,
		SilenceUsage:      true,
	}
	peers.AddCommand(
		cli.PeerAddCmd(),
		cli.PeerListCmd(),
		cli.PeerInfoCmd(),
		cli.PeerSetStatusCmd(),
	)
	servers.AddCommand(
		cli.InitServer(),
		cli.ListServer(),
		cli.DeleteServer(),
		cli.NewCipherKey(),
	)
	configure.AddCommand(
		cli.ConfigureSystemdServerCmd(),
		cli.ConfigureSystemdPeerCmd(),
		cli.ConfigureServerCmd(),
		cli.ConfigurePeersCmd(),
	)
	root.AddCommand(
		servers,
		peers,
		configure,
		cli.RunWebServerCmd(),
	)
	root.PersistentFlags().BoolVar(&cli.O.Local, "local", false, "Fetch from local database instead of remote.")
	root.PersistentFlags().BoolVar(&cli.O.ShowVersionAndExit, "version", false, "Show version.")
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
