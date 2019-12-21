package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// PersistentPreRunE will execute on every subcommand call
func PersistentPreRunE(cmd *cobra.Command, args []string) error {
	if O.Local {
		GlobalBoltOptions = InitEmptyBoltOptions()
	}
	if _, err := os.Stat(GlobalWGAppConfigPath); !os.IsNotExist(err) {
		return nil
	}
	return os.Mkdir(GlobalWGAppConfigPath, 0755)
}
