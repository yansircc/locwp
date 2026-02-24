package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yansircc/locwp/internal/exec"
	"github.com/yansircc/locwp/internal/site"
)

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a WordPress site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sc, err := site.LoadByName(args[0])
		if err != nil {
			return err
		}

		if err := exec.RunPawlWorkflow(sc.SiteDir, "stop"); err != nil {
			return err
		}

		fmt.Printf("Site %q stopped\n", sc.Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
