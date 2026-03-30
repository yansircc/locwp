package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <port>",
	Aliases: []string{"rm"},
	Short:   "Delete a WordPress site",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port %q: %w", args[0], err)
		}
		sc, err := site.LoadByPort(port)
		if err != nil {
			return err
		}

		// Run destroy workflow (removes Caddy/FPM configs, reloads Caddy)
		_ = exec.RunInDir(sc.SiteDir, "pawl", "start", "destroy")

		// Remove site directory
		os.RemoveAll(sc.SiteDir)

		fmt.Printf("Site %d deleted.\n", sc.Port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
