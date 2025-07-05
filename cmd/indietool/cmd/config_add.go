/*
Copyright © 2025
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// configAddCmd represents the config add command
var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add configuration settings",
	Long: `Add new configuration settings to your indietool configuration.

Currently supports adding provider configurations.

Examples:
  indietool config add provider cloudflare --api-key KEY --email EMAIL
  indietool config add provider porkbun --api-key KEY --api-secret SECRET`,
}

func init() {
	configCmd.AddCommand(configAddCmd)
}

