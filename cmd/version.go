package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is overridden at build time via
// -ldflags "-X github.com/thisisbalu/nemi/cmd.version=v1.2.3" (how GoReleaser
// stamps release builds). When that ldflag isn't set — e.g. `go install
// github.com/thisisbalu/nemi@v0.1.0` — resolveVersion falls back to the module
// version Go records in the binary's build metadata.
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Nemi version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("nemi %s\n", resolveVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
