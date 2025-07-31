/*
Copyright © 2025
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// dnsCmd represents the dns command
var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS records for your domains",
	Long: `Manage DNS records for your domains across different DNS providers.
Supports listing, setting, and deleting DNS records with automatic provider detection.

Examples:
  indietool dns list example.com
  indietool dns set example.com www A 192.168.1.1
  indietool dns delete example.com www A
  indietool dns set example.com @ MX "10 mail.example.com" --priority 10`,
}

func init() {
	rootCmd.AddCommand(dnsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dnsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dnsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
