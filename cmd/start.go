package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sc, err := site.LoadByName(args[0])
		if err != nil {
			return err
		}

		if err := exec.RunInDir(sc.SiteDir, "pawl", "start", "start"); err != nil {
			return err
		}

		fmt.Printf("Site %q started at http://localhost:%d (PHP %s)\n", sc.Name, sc.Port, sc.PHP)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
