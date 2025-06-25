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
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		enabledRegistrars := cfg.GetEnabledRegistrars()
		if len(enabledRegistrars) == 0 {
			fmt.Println("No registrars are enabled in the configuration")
		} else {
			fmt.Printf("Enabled registrars: %v\n", enabledRegistrars)
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
