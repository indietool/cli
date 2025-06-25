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

Currently supports adding registrar configurations.

Examples:
  indietool config add registrar cloudflare --api-key KEY --email EMAIL
  indietool config add registrar porkbun --api-key KEY --api-secret SECRET`,
}

func init() {
	configCmd.AddCommand(configAddCmd)
}