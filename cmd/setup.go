package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

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

		// Ensure php is linked (php@x.y is keg-only)
		if !exec.CommandExists("php") {
			fmt.Printf("  ... Linking %s...\n", phpFormula)
			_ = exec.Run("brew", "link", "--force", "--overwrite", phpFormula)
		}

		// --- Phase 1: Start user-level services (no sudo) ---
		fmt.Println("\nStarting user-level services...")
		_ = exec.Run("brew", "services", "restart", "mariadb")
		_ = exec.Run("brew", "services", "restart", phpFormula)

		// Wait for MariaDB to be ready (socket may take a few seconds)
		fmt.Print("  Waiting for MariaDB...")
		for i := 0; i < 30; i++ {
			if _, err := exec.Output("mariadb", "-e", "SELECT 1"); err == nil {
				break
			}
			fmt.Print(".")
			time.Sleep(time.Second)
		}
		fmt.Println(" ready")

		// Configure dnsmasq: .loc.wp domains → 127.0.0.1
		// dnsmasq uses default port 53 and runs as root (configured in Phase 2)
		fmt.Println("\nConfiguring dnsmasq...")
		dnsmasqConf := filepath.Join(template.HomebrewPrefix(), "etc", "dnsmasq.conf")
		confData, _ := os.ReadFile(dnsmasqConf)
		confStr := string(confData)
		dnsmasqLine := "address=/.loc.wp/127.0.0.1"
		// Check each line to avoid matching commented-out lines
		var found bool
		for _, l := range strings.Split(confStr, "\n") {
			if strings.TrimSpace(l) == dnsmasqLine {
				found = true
				break
			}
		}
		if !found {
			f, err := os.OpenFile(dnsmasqConf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open dnsmasq.conf: %w", err)
			}
			if _, err := f.WriteString("\n" + dnsmasqLine + "\n"); err != nil {
				f.Close()
				return fmt.Errorf("failed to write dnsmasq.conf: %w", err)
			}
			f.Close()
			fmt.Println("  [ok] dnsmasq configured (.loc.wp → 127.0.0.1)")
		} else {
			fmt.Println("  [ok] dnsmasq already configured")
		}

		// Install mkcert CA and generate wildcard certificate
		fmt.Println("\nSetting up SSL certificates...")
		if err := exec.Run("mkcert", "-install"); err != nil {
			return fmt.Errorf("mkcert -install failed: %w", err)
		}
		fmt.Println("  [ok] mkcert CA installed")

		sslDir := config.SSLDir()
		os.MkdirAll(sslDir, 0755)
		certFile := filepath.Join(sslDir, "_wildcard.loc.wp.pem")
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			if err := exec.RunInDir(sslDir, "mkcert", "-cert-file", "_wildcard.loc.wp.pem", "-key-file", "_wildcard.loc.wp-key.pem", "*.loc.wp"); err != nil {
				return fmt.Errorf("mkcert wildcard cert generation failed: %w", err)
			}
			fmt.Println("  [ok] wildcard certificate generated")
		} else {
			fmt.Println("  [ok] wildcard certificate already exists")
		}

		// --- Phase 2: Operations requiring sudo (grouped at the end) ---
		fmt.Println("\nAdmin privileges required for the following operations...")

		// Create /etc/resolver/wp for macOS DNS resolution
		resolverDir := "/etc/resolver"
		resolverFile := filepath.Join(resolverDir, "wp")
		if _, err := os.Stat(resolverFile); os.IsNotExist(err) {
			if err := exec.Run("sudo", "mkdir", "-p", resolverDir); err != nil {
				return fmt.Errorf("failed to create resolver dir: %w", err)
			}
			if err := exec.Run("sudo", "bash", "-c", fmt.Sprintf("printf 'nameserver 127.0.0.1\\n' > %s", resolverFile)); err != nil {
				return fmt.Errorf("failed to create resolver file: %w", err)
			}
			fmt.Println("  [ok] resolver configured")
		} else {
			fmt.Println("  [ok] resolver already configured")
		}

		// Start dnsmasq as system service (needs root for port 53)
		if err := exec.Run("sudo", "brew", "services", "restart", "dnsmasq"); err != nil {
			return fmt.Errorf("failed to start dnsmasq: %w", err)
		}
		fmt.Println("  [ok] dnsmasq started (system service)")

		// Write a clean nginx.conf (worker runs as current user to access FPM sockets/logs)
		currentUser, _ := user.Current()
		nginxConf := filepath.Join(template.HomebrewPrefix(), "etc", "nginx", "nginx.conf")
		nginxTemplate := fmt.Sprintf(`user  %s staff;
worker_processes  1;

events {
    worker_connections  1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile        on;
    keepalive_timeout  65;
    include servers/*;
}
`, currentUser.Username)
		os.WriteFile(nginxConf, []byte(nginxTemplate), 0644)
		fmt.Printf("  [ok] nginx configured (user: %s)\n", currentUser.Username)

		// Remove stale locwp symlinks from nginx servers dir (broken links crash nginx)
		nginxServersDir := filepath.Join(template.HomebrewPrefix(), "etc", "nginx", "servers")
		if entries, err := os.ReadDir(nginxServersDir); err == nil {
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), "locwp-") {
					link := filepath.Join(nginxServersDir, e.Name())
					if target, err := os.Readlink(link); err == nil {
						if _, err := os.Stat(target); os.IsNotExist(err) {
							os.Remove(link)
						}
					}
				}
			}
		}

		// Allow passwordless sudo for nginx reload (so locwp add doesn't prompt)
		nginxBin, _ := exec.Output("which", "nginx")
		nginxBin = strings.TrimSpace(nginxBin)
		sudoersFile := "/etc/sudoers.d/locwp"
		sudoersLine := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: %s\n", currentUser.Username, nginxBin)
		if _, err := os.Stat(sudoersFile); os.IsNotExist(err) {
			if err := exec.Run("sudo", "bash", "-c", fmt.Sprintf("printf '%s' > %s && chmod 0440 %s", sudoersLine, sudoersFile, sudoersFile)); err != nil {
				return fmt.Errorf("failed to configure sudoers: %w", err)
			}
			fmt.Println("  [ok] passwordless nginx reload configured")
		} else {
			fmt.Println("  [ok] passwordless nginx reload already configured")
		}

		// Start nginx (needs root for ports 80/443)
		// Use direct `sudo nginx` instead of brew services so nginx runs as daemon
		// and writes a PID file, enabling `nginx -s reload` for vhost updates.
		_ = exec.Run("sudo", "brew", "services", "stop", "nginx")
		_ = exec.Run("sudo", "nginx", "-s", "quit")
		_ = exec.Run("sudo", "pkill", "-9", "nginx")
		time.Sleep(2 * time.Second)
		if err := exec.Run("sudo", "nginx"); err != nil {
			return fmt.Errorf("failed to start nginx: %w", err)
		}
		fmt.Println("  [ok] nginx started")

		fmt.Println("\nSetup complete.")
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&flagSetupPHP, "php", config.DefaultPHP, "PHP version to install (e.g. 8.1, 8.2, 8.3)")
	rootCmd.AddCommand(setupCmd)
}
