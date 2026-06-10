package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thisisbalu/nemi/internal/check"
)

var checkDrafts bool

var checkCmd = &cobra.Command{
	Use:           "check",
	Short:         "Check content for broken links and frontmatter issues",
	RunE:          runCheck,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVarP(&checkDrafts, "drafts", "D", false, "include draft pages")
}

func runCheck(_ *cobra.Command, _ []string) error {
	rep, err := check.Run("content", "static", checkDrafts)
	if err != nil {
		return err
	}

	for _, is := range rep.Issues {
		label := "warning"
		if is.IsError {
			label = "error"
		}
		fmt.Printf("  %-7s %s: %s\n", label, is.Page, is.Message)
	}

	errs, warns := rep.Errors(), rep.Warnings()
	if len(rep.Issues) == 0 {
		fmt.Println("✓ No issues found")
		return nil
	}
	fmt.Printf("\n%d error(s), %d warning(s)\n", errs, warns)
	if errs > 0 {
		return fmt.Errorf("check failed")
	}
	return nil
}
