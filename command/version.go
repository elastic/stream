package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "UNKNOWN"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stream %s\n", version)
	},
}
