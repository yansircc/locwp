package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"rm"},
	Short:   "Delete a WordPress site and its database",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		sc, err := site.LoadByName(name)
		if err != nil {
			return err
		}

		// Run destroy workflow (drops DB, removes nginx/FPM configs, reloads nginx)
		_ = exec.RunPawlWorkflow(sc.SiteDir, "destroy")

		// Remove site directory (pawl can't delete its own working directory)
		os.RemoveAll(sc.SiteDir)

		fmt.Printf("Site %q deleted.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
