package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var wpCmd = &cobra.Command{
	Use:                "wp <port> -- <wp-cli args...>",
	Short:              "Run WP-CLI commands for a site",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port %q: %w", args[0], err)
		}
		sc, err := site.LoadByPort(port)
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
