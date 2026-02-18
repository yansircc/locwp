package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "locwp",
	Short: "Local WordPress site manager",
	Long:  "Create and manage local WordPress development sites using native PHP, MariaDB, and Nginx.",
}

func Execute() error {
	return rootCmd.Execute()
}
