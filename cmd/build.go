package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thisisbalu/nemi/internal/build"
)

var buildDrafts bool

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the site into public/",
	RunE: func(_ *cobra.Command, _ []string) error {
		start := time.Now()
		stats, err := build.Run(false, buildDrafts)
		if err != nil {
			return err
		}
		fmt.Printf("Built %d pages in %s → public/\n", stats.Pages, time.Since(start).Round(time.Millisecond))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolVarP(&buildDrafts, "drafts", "D", false, "include draft pages")
}
