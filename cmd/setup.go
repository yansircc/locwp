package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/template"
)

var flagSetupPHP string

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install dependencies (PHP, MariaDB, Nginx, WP-CLI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		phpFormula := template.PHPFormulaName(flagSetupPHP)

		deps := []struct {
			name    string
			check   string
			install string
		}{
			{phpFormula, "", "brew install " + phpFormula},
			{"mariadb", "mariadb", "brew install mariadb"},
			{"nginx", "nginx", "brew install nginx"},
			{"wp-cli", "wp", "brew install wp-cli"},
		}

		for _, d := range deps {
			if d.check != "" && exec.CommandExists(d.check) {
				fmt.Printf("  [ok] %s already installed\n", d.name)
				continue
			}
			// For PHP, check if the formula is already installed via brew
			if d.check == "" {
				installed, _ := exec.Output("brew", "list", "--formula", d.name)
				if strings.TrimSpace(installed) != "" {
					fmt.Printf("  [ok] %s already installed\n", d.name)
					continue
				}
			}
			fmt.Printf("  ... Installing %s...\n", d.name)
			if err := exec.Run("bash", "-c", d.install); err != nil {
				return fmt.Errorf("failed to install %s: %w", d.name, err)
			}
			fmt.Printf("  [ok] %s installed\n", d.name)
		}

		// Ensure services are running
		fmt.Println("\nStarting services...")
		_ = exec.Run("brew", "services", "start", "mariadb")
		_ = exec.Run("brew", "services", "start", phpFormula)
		_ = exec.Run("brew", "services", "start", "nginx")

		fmt.Println("\nSetup complete.")
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&flagSetupPHP, "php", "8.3", "PHP version to install (e.g. 8.1, 8.2, 8.3)")
	rootCmd.AddCommand(setupCmd)
}
