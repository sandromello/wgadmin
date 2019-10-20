package main

import (
	"os"

	"github.com/sandromello/wgadmin/pkg/cli"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "wgapp",
		Short: "wgapp manages users and wireguard servers.",
		Run: func(cmd *cobra.Command, args []string) {
			if cli.O.ShowVersionAndExit {
				// version.PrintAndExit()
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
	)
	configure.AddCommand(
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
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
