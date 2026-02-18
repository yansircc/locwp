package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/site"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all local WordPress sites",
	RunE: func(cmd *cobra.Command, args []string) error {
		sitesDir := filepath.Join(config.BaseDir(), "sites")
		entries, err := os.ReadDir(sitesDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No sites yet. Run `locwp add <name>` to create one.")
				return nil
			}
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDOMAIN\tPHP\tSTATUS")
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			sc, err := site.Load(filepath.Join(sitesDir, e.Name()))
			if err != nil {
				fmt.Fprintf(w, "%s\t-\t-\terror\n", e.Name())
				continue
			}
			status := site.Status(sc)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", sc.Name, sc.Domain, sc.PHP, status)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
