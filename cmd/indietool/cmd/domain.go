/*
Copyright © 2025
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// domainCmd represents the domain command
var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Domain discovery and availability checking",
	Long: `Search for domain availability and explore domain names across multiple TLDs.
Uses RDAP (Registration Data Access Protocol) for reliable domain status checking.

This command provides domain discovery tools for indie hackers and startups to find
available domain names. It supports checking individual domains or exploring a base
name across popular TLDs.

Available subcommands:
  search   Check availability of specific domain names
  explore  Explore a domain name across multiple popular TLDs

Examples:
  indietool domain search example.com
  indietool domain explore myapp
  indietool domain explore startup --tlds com,org,dev,ai

The domain command also shows your current configuration status including
enabled registrars and configuration validation results.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Send metrics for domain command and subcommands
		if metricsAgent := GetMetricsAgent(); metricsAgent != nil {
			commandName := "domain " + cmd.Name()
			metadata := make(map[string]string)

			// No specific metadata needed for domain commands yet
			// Future: could track provider info if domain commands get provider-specific features

			// Track command execution asynchronously
			PendingItems(metricsAgent.Observe(commandName, args, metadata, 0))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Example of using the global config
		cfg := GetConfig()
		if cfg == nil {
			fmt.Println("No configuration available")
			return
		}

		if !cfg.Valid() {
			fmt.Println("No valid configuration loaded - check your config file")
			return
		}

		// Show enabled registrars
		enabledProviders := cfg.GetEnabledProviders()
		if len(enabledProviders) == 0 {
			fmt.Println("No registrars are enabled in the configuration")
		} else {
			fmt.Printf("Enabled registrars: %v\n", enabledProviders)
		}

		// Show configuration validation status
		if errors := cfg.ValidateConfig(); len(errors) > 0 {
			fmt.Printf("Configuration validation issues: %v\n", errors)
		} else {
			fmt.Println("Configuration is valid")
		}
	},
}

func init() {
	rootCmd.AddCommand(domainCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// domainCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// domainCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
