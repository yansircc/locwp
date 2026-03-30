package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new local WordPress site",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		baseDir := config.BaseDir()

		// Allocate next available port
		port := config.NextPort(baseDir)
		portStr := strconv.Itoa(port)
		siteDir := filepath.Join(baseDir, "sites", portStr)

		if _, err := os.Stat(siteDir); err == nil {
			return fmt.Errorf("site %s already exists", portStr)
		}

		sc := &site.Config{
			Port:       port,
			PHP:        flagPHP,
			WPVer:      "latest",
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

		// Generate Caddy site config
		caddySitesDir := config.CaddySitesDir()
		if err := os.MkdirAll(caddySitesDir, 0755); err != nil {
			return err
		}
		caddyConfPath := filepath.Join(caddySitesDir, portStr+".caddy")
		if err := template.WriteCaddyConf(caddyConfPath, sc); err != nil {
			return err
		}

		// Generate PHP-FPM pool (local copy)
		phpDir := filepath.Join(baseDir, "php")
		if err := os.MkdirAll(phpDir, 0755); err != nil {
			return err
		}
		if err := template.WriteFPMPool(filepath.Join(phpDir, portStr+".conf"), sc); err != nil {
			return err
		}

		// Install pool config into Homebrew PHP-FPM pool.d
		fpmPoolDir := template.FPMPoolDir(sc.PHP)
		if _, err := os.Stat(fpmPoolDir); err == nil {
			if err := template.WriteFPMPool(filepath.Join(fpmPoolDir, "locwp-"+portStr+".conf"), sc); err != nil {
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

		fmt.Printf("Site configured (%s, PHP %s)\n", sc.URL(), flagPHP)

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
