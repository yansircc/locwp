package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var wpCmd = &cobra.Command{
	Use:                "wp <name> -- <wp-cli args...>",
	Short:              "Run WP-CLI commands for a site",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		sc, err := site.LoadByName(name)
		if err != nil {
			return err
		}

		// Find "--" separator
		wpArgs := []string{"--path=" + sc.WPRoot}
		for i, a := range args[1:] {
			if a == "--" {
				wpArgs = append(wpArgs, args[i+2:]...)
				break
			}
		}

		return exec.Run("wp", wpArgs...)
	},
}

func init() {
	rootCmd.AddCommand(wpCmd)
}
