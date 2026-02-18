package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sc, err := site.LoadByName(args[0])
		if err != nil {
			return err
		}

		if err := site.DisableVhost(sc.Name); err != nil {
			return fmt.Errorf("failed to disable vhost: %w", err)
		}

		if err := exec.Run("nginx", "-s", "reload"); err != nil {
			return fmt.Errorf("nginx reload failed: %w", err)
		}

		fmt.Printf("Site %q stopped\n", sc.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
