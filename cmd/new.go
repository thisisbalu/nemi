package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thisisbalu/nemi/internal/content"
	"github.com/thisisbalu/nemi/scaffold"
)

var newCmd = &cobra.Command{
	Use:   "new <site-name>",
	Short: "Create a new Nemi site",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

var newPostSection string

var newPostCmd = &cobra.Command{
	Use:          "post <title>",
	Short:        "Scaffold a new draft post",
	Args:         cobra.MinimumNArgs(1),
	RunE:         runNewPost,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.AddCommand(newPostCmd)
	newPostCmd.Flags().StringVarP(&newPostSection, "section", "s", "blog", "content section for the post")
}

func runNewPost(_ *cobra.Command, args []string) error {
	title := strings.Join(args, " ")
	slug := content.Urlize(title)
	if slug == "" {
		return fmt.Errorf("could not derive a slug from %q", title)
	}
	path := filepath.Join("content", newPostSection, slug+".md")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	body := fmt.Sprintf("---\ntitle: %q\ndate: %s\ndraft: true\n---\n\n", title, time.Now().Format("2006-01-02"))
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", path)
	return nil
}

func runNew(_ *cobra.Command, args []string) error {
	name := args[0]
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}
	if err := copyScaffold(name); err != nil {
		return err
	}
	fmt.Printf("Created %s\n\n", name)
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  nemi serve")
	return nil
}

func copyScaffold(dest string) error {
	return fs.WalkDir(scaffold.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := scaffold.FS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
