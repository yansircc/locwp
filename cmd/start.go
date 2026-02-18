package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
	"github.com/yansircc/locwp/internal/template"
)

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sc, err := site.LoadByName(args[0])
		if err != nil {
			return err
		}

		if err := site.EnableVhost(sc.Name); err != nil {
			return fmt.Errorf("failed to enable vhost: %w", err)
		}

		// Ensure the correct PHP-FPM version is running
		phpFormula := template.PHPFormulaName(sc.PHP)
		_ = exec.Run("brew", "services", "start", phpFormula)

		if err := exec.Run("nginx", "-s", "reload"); err != nil {
			return fmt.Errorf("nginx reload failed: %w", err)
		}

		fmt.Printf("Site %q started at http://localhost:%d (PHP %s)\n", sc.Name, sc.Port, sc.PHP)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
