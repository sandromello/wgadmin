package cli

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func roundTime(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

func getDeltaDuration(startTime, endTime string) string {
	start, _ := time.Parse(time.RFC3339, startTime)
	end, _ := time.Parse(time.RFC3339, endTime)
	delta := end.Sub(start)
	var d time.Duration
	if endTime != "" {
		d = roundTime(delta, time.Second)
	} else {
		d = roundTime(time.Since(start), time.Second)
	}
	switch {
	case d.Hours() >= 24: // day resolution
		return fmt.Sprintf("%.fd", math.Floor(d.Hours()/24))
	case d.Hours() >= 8760: // year resolution
		return fmt.Sprintf("%.fd", math.Floor(d.Hours()/8760))
	}
	return d.String()
}

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
