package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var startCmd = &cobra.Command{
	Use:   "start <port>",
	Short: "Start a WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port %q: %w", args[0], err)
		}
		sc, err := site.LoadByPort(port)
		if err != nil {
			return err
		}

		if err := exec.RunInDir(sc.SiteDir, "pawl", "start", "--reset", "start"); err != nil {
			return err
		}

		fmt.Printf("Site started at %s (PHP %s)\n", sc.URL(), sc.PHP)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
