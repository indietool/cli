package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// onOff renders a boolean as on/off.
func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// domainGetCmd represents the domain get command
var domainGetCmd = &cobra.Command{
	Use:   "get <domain>",
	Short: "Show full details for a managed domain",
	Long: `Show the full registrar details for a domain: expiry, auto-renewal, registrar
lock, WHOIS privacy, nameservers, and renewal price.

Examples:
  indietool domain get example.dev
  indietool domain get example.dev --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))

		_, dm, err := findRegistrarForDomain(cmd.Context(), name)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if jsonOutput {
			data, err := json.MarshalIndent(dm, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal domain: %w", err)
			}
			fmt.Fprintln(out, string(data))
			return nil
		}

		fmt.Fprintf(out, "Domain:      %s\n", dm.Name)
		fmt.Fprintf(out, "Provider:    %s\n", dm.Provider)
		fmt.Fprintf(out, "Status:      %s\n", dm.Status)
		fmt.Fprintf(out, "Expires:     %s\n", dm.ExpiryDate.Format("2006-01-02"))
		fmt.Fprintf(out, "Auto-renew:  %s\n", onOff(dm.AutoRenewal))
		fmt.Fprintf(out, "Locked:      %s\n", onOff(dm.Locked))
		fmt.Fprintf(out, "Privacy:     %s\n", onOff(dm.Privacy))
		if len(dm.Nameservers) > 0 {
			fmt.Fprintf(out, "Nameservers: %s\n", strings.Join(dm.Nameservers, ", "))
		}
		if dm.Cost != nil {
			fmt.Fprintf(out, "Renewal:     %.2f %s\n", dm.Cost.RenewalPrice, dm.Cost.Currency)
		}
		return nil
	},
}

func init() {
	domainCmd.AddCommand(domainGetCmd)
}
