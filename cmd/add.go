package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
	"github.com/yansircc/locwp/internal/template"
)

var (
	flagPHP        string
	flagNoStart    bool
	flagAdminUser  string
	flagAdminPass  string
	flagAdminEmail string
)

var namePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new local WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Validate site name
		if !namePattern.MatchString(name) {
			return fmt.Errorf("invalid site name %q: only lowercase letters, digits, and hyphens allowed (cannot start/end with hyphen)", name)
		}

		baseDir := config.BaseDir()
		siteDir := filepath.Join(baseDir, "sites", name)

		if _, err := os.Stat(siteDir); err == nil {
			return fmt.Errorf("site %q already exists", name)
		}

		// Generate domain and check for duplicates
		domain := name + ".loc.wp"
		if config.DomainExists(baseDir, domain) {
			return fmt.Errorf("domain %q is already in use", domain)
		}

		// DB user: current system user (Homebrew MariaDB default)
		dbUser := "root"
		if u, err := user.Current(); err == nil {
			dbUser = u.Username
		}

		sc := &site.Config{
			Name:       name,
			Domain:     domain,
			PHP:        flagPHP,
			WPVer:      "latest",
			DBName:     "wp_" + strings.ReplaceAll(name, "-", "_"),
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

		// Generate pawl workflows
		workflowDir := filepath.Join(siteDir, ".pawl", "workflows")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			return err
		}
		if err := template.WritePawlWorkflows(workflowDir, sc); err != nil {
			return err
		}

		fmt.Printf("Site %q configured (https://%s, PHP %s)\n", name, domain, flagPHP)

		if flagNoStart {
			return nil
		}

		// Run provision workflow
		return exec.RunInDir(siteDir, "pawl", "start", "provision")
	},
}

func init() {
	addCmd.Flags().StringVar(&flagPHP, "php", config.DefaultPHP, "PHP version")
	addCmd.Flags().BoolVar(&flagNoStart, "no-start", false, "Don't start provisioning immediately")
	addCmd.Flags().StringVar(&flagAdminUser, "user", "admin", "WordPress admin username")
	addCmd.Flags().StringVar(&flagAdminPass, "pass", "admin", "WordPress admin password")
	addCmd.Flags().StringVar(&flagAdminEmail, "email", "admin@loc.wp", "WordPress admin email")
	rootCmd.AddCommand(addCmd)
}
