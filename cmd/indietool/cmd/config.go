/*
Copyright © 2025
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage indietool configuration",
	Long: `Manage indietool configuration settings including provider credentials,
domain management preferences, and other application settings.

Examples:
  indietool config add provider cloudflare --api-key KEY --email EMAIL
  indietool config add provider porkbun --api-key KEY --api-secret SECRET`,
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
