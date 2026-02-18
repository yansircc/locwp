package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
	"github.com/yansircc/locwp/internal/template"
)

var (
	flagPort       int
	flagPHP        string
	flagNoStart    bool
	flagAdminUser  string
	flagAdminPass  string
	flagAdminEmail string
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new local WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		baseDir := config.BaseDir()
		siteDir := filepath.Join(baseDir, "sites", name)

		if _, err := os.Stat(siteDir); err == nil {
			return fmt.Errorf("site %q already exists", name)
		}

		// Port: use flag or auto-assign
		port := flagPort
		if port == 0 {
			p, err := config.NextPort(baseDir)
			if err != nil {
				return err
			}
			port = p
		}

		// DB user: current system user (Homebrew MariaDB default)
		dbUser := "root"
		if u, err := user.Current(); err == nil {
			dbUser = u.Username
		}

		sc := &site.Config{
			Name:       name,
			Port:       port,
			PHP:        flagPHP,
			WPVer:      "latest",
			DBName:     "wp_" + name,
			DBUser:     dbUser,
			DBHost:     "localhost",
			SiteDir:    siteDir,
			WPRoot:     filepath.Join(siteDir, "wordpress"),
			AdminUser:  flagAdminUser,
			AdminPass:  flagAdminPass,
			AdminEmail: flagAdminEmail,
		}

		// Create directories
		for _, d := range []string{sc.WPRoot, filepath.Join(siteDir, "logs")} {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", d, err)
			}
		}

		// Save site config
		if err := site.Save(siteDir, sc); err != nil {
			return err
		}

		// Generate nginx vhost
		nginxDir := filepath.Join(baseDir, "nginx", "sites")
		if err := os.MkdirAll(nginxDir, 0755); err != nil {
			return err
		}
		vhostPath := filepath.Join(nginxDir, name+".conf")
		if err := template.WriteNginxConf(vhostPath, sc); err != nil {
			return err
		}

		// Symlink vhost into Homebrew nginx servers dir
		nginxServersDir := filepath.Join(template.HomebrewPrefix(), "etc", "nginx", "servers")
		os.MkdirAll(nginxServersDir, 0755)
		linkPath := filepath.Join(nginxServersDir, "locwp-"+name+".conf")
		os.Remove(linkPath)
		os.Symlink(vhostPath, linkPath)

		// Generate PHP-FPM pool (local copy)
		phpDir := filepath.Join(baseDir, "php")
		if err := os.MkdirAll(phpDir, 0755); err != nil {
			return err
		}
		if err := template.WriteFPMPool(filepath.Join(phpDir, name+".conf"), sc); err != nil {
			return err
		}

		// Install pool config into Homebrew PHP-FPM pool.d
		fpmPoolDir := template.FPMPoolDir(sc.PHP)
		if _, err := os.Stat(fpmPoolDir); err == nil {
			if err := template.WriteFPMPool(filepath.Join(fpmPoolDir, "locwp-"+name+".conf"), sc); err != nil {
				return fmt.Errorf("write FPM pool to %s: %w", fpmPoolDir, err)
			}
		}

		// Generate pawl workflow
		pawlDir := filepath.Join(siteDir, ".pawl")
		if err := os.MkdirAll(pawlDir, 0755); err != nil {
			return err
		}
		if err := template.WritePawlConfig(filepath.Join(pawlDir, "config.json"), sc); err != nil {
			return err
		}

		fmt.Printf("Site %q configured (port %d, PHP %s)\n", name, port, flagPHP)

		if flagNoStart {
			return nil
		}

		// Run pawl workflow
		return exec.RunInDir(siteDir, "pawl", "start", name)
	},
}

func init() {
	addCmd.Flags().IntVar(&flagPort, "port", 0, "Port number (auto-assigned from 8081)")
	addCmd.Flags().StringVar(&flagPHP, "php", "8.3", "PHP version")
	addCmd.Flags().BoolVar(&flagNoStart, "no-start", false, "Don't start provisioning immediately")
	addCmd.Flags().StringVar(&flagAdminUser, "user", "admin", "WordPress admin username")
	addCmd.Flags().StringVar(&flagAdminPass, "pass", "admin", "WordPress admin password")
	addCmd.Flags().StringVar(&flagAdminEmail, "email", "admin@local.test", "WordPress admin email")
	rootCmd.AddCommand(addCmd)
}
