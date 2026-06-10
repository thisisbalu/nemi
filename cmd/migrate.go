package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thisisbalu/nemi/internal/migrate"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate a Hugo, Jekyll, or raw HTML site to Nemi",
}

var migrateHugoCmd = &cobra.Command{
	Use:   "hugo <source> [dest]",
	Short: "Migrate a Hugo site to Nemi",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMigrateHugo,
}

var migrateJekyllCmd = &cobra.Command{
	Use:   "jekyll <source> [dest]",
	Short: "Migrate a Jekyll site to Nemi",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMigrateJekyll,
}

var migrateHTMLCmd = &cobra.Command{
	Use:   "html <source> [dest]",
	Short: "Migrate a folder of raw HTML pages to Nemi",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMigrateHTML,
}

func init() {
	migrateCmd.AddCommand(migrateHugoCmd)
	migrateCmd.AddCommand(migrateJekyllCmd)
	migrateCmd.AddCommand(migrateHTMLCmd)
	rootCmd.AddCommand(migrateCmd)
}

func runMigrateHugo(_ *cobra.Command, args []string) error {
	src, dst := migratePaths(args)
	fmt.Printf("Migrating Hugo site: %s → %s\n\n", src, dst)
	result, err := migrate.Hugo(src, dst)
	if err != nil {
		return err
	}
	printMigrateResult(result, dst)
	return nil
}

func runMigrateJekyll(_ *cobra.Command, args []string) error {
	src, dst := migratePaths(args)
	fmt.Printf("Migrating Jekyll site: %s → %s\n\n", src, dst)
	result, err := migrate.Jekyll(src, dst)
	if err != nil {
		return err
	}
	printMigrateResult(result, dst)
	return nil
}

func runMigrateHTML(_ *cobra.Command, args []string) error {
	src, dst := migratePaths(args)
	fmt.Printf("Migrating HTML pages: %s → %s\n\n", src, dst)
	result, err := migrate.HTML(src, dst)
	if err != nil {
		return err
	}
	printMigrateResult(result, dst)
	return nil
}

func migratePaths(args []string) (src, dst string) {
	src = args[0]
	if len(args) > 1 {
		dst = args[1]
	} else {
		dst = filepath.Base(src) + "-nemi"
	}
	return
}

func printMigrateResult(r *migrate.Result, dst string) {
	fmt.Printf("  Config:  nemi.toml\n")
	fmt.Printf("  Content: %d pages\n", r.Pages)
	fmt.Printf("  Static:  %d files\n", r.Static)
	if len(r.Warnings) > 0 {
		fmt.Printf("\n  Warnings (%d):\n", len(r.Warnings))
		for _, w := range r.Warnings {
			fmt.Printf("    ! %s\n", w)
		}
	}
	fmt.Printf("\nDone. Run:\n  cd %s\n  nemi serve\n", filepath.Base(dst))
}
