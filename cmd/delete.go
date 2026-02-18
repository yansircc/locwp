package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
	"github.com/yansircc/locwp/internal/template"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"rm"},
	Short:   "Delete a WordPress site and its database",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		sc, err := site.LoadByName(name)
		if err != nil {
			return err
		}

		baseDir := config.BaseDir()

		// Drop database
		_ = exec.Run("mariadb", "-u", sc.DBUser, "-e", fmt.Sprintf("DROP DATABASE IF EXISTS %s", sc.DBName))

		// Remove nginx vhost and symlink
		os.Remove(filepath.Join(baseDir, "nginx", "sites", name+".conf"))
		os.Remove(filepath.Join(baseDir, "nginx", "sites", name+".conf.disabled"))
		os.Remove(filepath.Join(template.HomebrewPrefix(), "etc", "nginx", "servers", "locwp-"+name+".conf"))

		// Remove FPM pool (local copy and Homebrew pool.d copy)
		os.Remove(filepath.Join(baseDir, "php", name+".conf"))
		os.Remove(filepath.Join(template.FPMPoolDir(sc.PHP), "locwp-"+name+".conf"))

		// Remove site directory
		os.RemoveAll(filepath.Join(baseDir, "sites", name))

		// Reload nginx
		_ = exec.Run("nginx", "-s", "reload")

		fmt.Printf("Site %q deleted.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
