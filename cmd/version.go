package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is overridden at build time via
// -ldflags "-X github.com/thisisbalu/nemi/cmd.version=v1.2.3".
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Nemi version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("nemi %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
