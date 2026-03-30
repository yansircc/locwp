package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/template"
)

var flagSetupPHP string

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install dependencies (PHP, Caddy, WP-CLI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		phpFormula := template.PHPFormulaName(flagSetupPHP)

		deps := []struct {
			name    string
			check   string
			install string
		}{
			{phpFormula, "", "brew install " + phpFormula},
			{"caddy", "caddy", "brew install caddy"},
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

		// Ensure php is linked (php@x.y is keg-only)
		if !exec.CommandExists("php") {
			fmt.Printf("  ... Linking %s...\n", phpFormula)
			_ = exec.Run("brew", "link", "--force", "--overwrite", phpFormula)
		}

		// Start user-level services (no sudo)
		fmt.Println("\nStarting services...")
		_ = exec.Run("brew", "services", "restart", phpFormula)

		// Configure Caddy
		fmt.Println("\nConfiguring Caddy...")
		caddySitesDir := config.CaddySitesDir()
		if err := os.MkdirAll(caddySitesDir, 0755); err != nil {
			return fmt.Errorf("failed to create caddy sites dir: %w", err)
		}

		// Write main Caddyfile that imports per-site configs
		caddyfilePath := filepath.Join(template.HomebrewPrefix(), "etc", "Caddyfile")
		caddyfileContent := fmt.Sprintf("{\n\tauto_https off\n}\n\nimport %s/*.caddy\n", caddySitesDir)
		if err := os.WriteFile(caddyfilePath, []byte(caddyfileContent), 0644); err != nil {
			return fmt.Errorf("failed to write Caddyfile: %w", err)
		}
		fmt.Println("  [ok] Caddyfile configured")

		// Start Caddy (user-level service, high ports only — no sudo)
		_ = exec.Run("brew", "services", "restart", "caddy")
		fmt.Println("  [ok] Caddy started")

		fmt.Println("\nSetup complete.")
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&flagSetupPHP, "php", config.DefaultPHP, "PHP version to install (e.g. 8.1, 8.2, 8.3)")
	rootCmd.AddCommand(setupCmd)
}
