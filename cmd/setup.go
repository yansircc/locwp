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
	Short: "Install dependencies (PHP, MariaDB, Nginx, WP-CLI, dnsmasq, mkcert)",
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
			{"dnsmasq", "dnsmasq", "brew install dnsmasq"},
			{"mkcert", "mkcert", "brew install mkcert"},
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

		// Configure dnsmasq for .local domains
		fmt.Println("\nConfiguring dnsmasq...")
		dnsmasqConf := filepath.Join(template.HomebrewPrefix(), "etc", "dnsmasq.conf")
		confData, _ := os.ReadFile(dnsmasqConf)
		dnsmasqLine := "address=/.local/127.0.0.1"
		if !strings.Contains(string(confData), dnsmasqLine) {
			f, err := os.OpenFile(dnsmasqConf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open dnsmasq.conf: %w", err)
			}
			if _, err := f.WriteString("\n" + dnsmasqLine + "\n"); err != nil {
				f.Close()
				return fmt.Errorf("failed to write dnsmasq.conf: %w", err)
			}
			f.Close()
			fmt.Println("  [ok] dnsmasq configured for .local domains")
		} else {
			fmt.Println("  [ok] dnsmasq already configured")
		}
		_ = exec.Run("brew", "services", "restart", "dnsmasq")

		// Create /etc/resolver/local for macOS DNS resolution
		fmt.Println("\nConfiguring macOS resolver...")
		resolverDir := "/etc/resolver"
		resolverFile := filepath.Join(resolverDir, "local")
		if _, err := os.Stat(resolverFile); os.IsNotExist(err) {
			fmt.Println("  Creating /etc/resolver/local (requires sudo)...")
			if err := exec.Run("sudo", "mkdir", "-p", resolverDir); err != nil {
				return fmt.Errorf("failed to create resolver dir: %w", err)
			}
			if err := exec.Run("sudo", "bash", "-c", "echo 'nameserver 127.0.0.1' > "+resolverFile); err != nil {
				return fmt.Errorf("failed to create resolver file: %w", err)
			}
			fmt.Println("  [ok] resolver configured")
		} else {
			fmt.Println("  [ok] resolver already configured")
		}

		// Install mkcert CA and generate wildcard certificate
		fmt.Println("\nSetting up SSL certificates...")
		if err := exec.Run("mkcert", "-install"); err != nil {
			return fmt.Errorf("mkcert -install failed: %w", err)
		}
		fmt.Println("  [ok] mkcert CA installed")

		sslDir := config.SSLDir()
		os.MkdirAll(sslDir, 0755)
		certFile := filepath.Join(sslDir, "_wildcard.local.pem")
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			if err := exec.RunInDir(sslDir, "mkcert", "-cert-file", "_wildcard.local.pem", "-key-file", "_wildcard.local-key.pem", "*.local"); err != nil {
				return fmt.Errorf("mkcert wildcard cert generation failed: %w", err)
			}
			fmt.Println("  [ok] wildcard certificate generated")
		} else {
			fmt.Println("  [ok] wildcard certificate already exists")
		}

		fmt.Println("\nSetup complete.")
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&flagSetupPHP, "php", "8.3", "PHP version to install (e.g. 8.1, 8.2, 8.3)")
	rootCmd.AddCommand(setupCmd)
}
