package main

import (
	"github.com/sandromello/wgadmin/pkg/cli"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "wgapp",
		Short: "wgapp manages users and wireguard servers.",
		// PostRun: cleanup,
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
		PersistentPreRunE: cli.CreateConfigPath,
		SilenceUsage:      true,
	}
	peers := &cobra.Command{
		Use:               "peer",
		Aliases:           []string{"peers"},
		Short:             "Interact with peer resources.",
		PersistentPreRunE: cli.CreateConfigPath,
		SilenceUsage:      true,
	}
	peers.AddCommand(
		cli.PeerAdd(),
		cli.PeerList(),
		cli.PeerInfo(),
		cli.PeerSetStatus(),
	)
	servers.AddCommand(
		cli.InitServer(),
		cli.ListServer(),
		cli.DeleteServer(),
	)
	root.AddCommand(servers, peers)
	root.Execute()
}
