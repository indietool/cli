package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/indietool/cli/domains"

	"github.com/spf13/cobra"
)

var (
	domainSetAutoRenew bool
	domainSetPrivacy   bool
	domainSetLocked    bool
	domainSetOn        bool
	domainSetOff       bool
)

// domainSetCmd represents the domain set command
var domainSetCmd = &cobra.Command{
	Use:   "set <domain>",
	Short: "Update domain settings (auto-renew; privacy/lock dashboard-only)",
	Long: `Update mutable registrar settings for a domain.

Currently only auto-renewal can be changed through the registrar APIs
(Cloudflare and Porkbun). Registrar lock and WHOIS privacy are not supported
by either API: passing --privacy or --locked fails fast with an error
instead of calling the API. Manage those settings in the provider's
dashboard (Cloudflare: Domains > Registrations; Porkbun: domain detail
page).

Examples:
  indietool domain set example.dev --auto-renew --on
  indietool domain set example.dev --auto-renew --off`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if domainSetOn && domainSetOff {
			return fmt.Errorf("--on and --off are mutually exclusive")
		}
		if !domainSetOn && !domainSetOff {
			return fmt.Errorf("specify --on or --off")
		}
		if !domainSetAutoRenew && !domainSetPrivacy && !domainSetLocked {
			return fmt.Errorf("specify at least one setting to change: --auto-renew, --privacy, --locked")
		}

		name := strings.ToLower(strings.TrimSpace(args[0]))

		registrar, _, err := findRegistrarForDomain(cmd.Context(), name)
		if err != nil {
			return err
		}

		manager, ok := domains.AsSettingsManager(registrar)
		if !ok {
			return fmt.Errorf("the registrar for %s does not support updating domain settings", name)
		}

		enabled := domainSetOn
		settings := domains.DomainSettings{}
		changed := []string{}
		if domainSetAutoRenew {
			settings.AutoRenew = &enabled
			changed = append(changed, "auto-renew")
		}
		if domainSetPrivacy {
			settings.Privacy = &enabled
			changed = append(changed, "privacy")
		}
		if domainSetLocked {
			settings.Locked = &enabled
			changed = append(changed, "locked")
		}

		if err := manager.UpdateDomainSettings(cmd.Context(), name, settings); err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if jsonOutput {
			data, err := json.MarshalIndent(map[string]any{
				"domain":   name,
				"settings": settings,
			}, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Fprintln(out, string(data))
			return nil
		}

		state := "on"
		if !enabled {
			state = "off"
		}
		fmt.Fprintf(out, "Updated %s for %s: set %s.\n", strings.Join(changed, ", "), name, state)
		return nil
	},
}

func init() {
	domainCmd.AddCommand(domainSetCmd)

	domainSetCmd.Flags().BoolVar(&domainSetAutoRenew, "auto-renew", false, "Change the auto-renewal setting")
	domainSetCmd.Flags().BoolVar(&domainSetPrivacy, "privacy", false, "Change the WHOIS privacy setting (not supported by the Cloudflare/Porkbun APIs; dashboard-only)")
	domainSetCmd.Flags().BoolVar(&domainSetLocked, "locked", false, "Change the registrar lock setting (not supported by the Cloudflare/Porkbun APIs; dashboard-only)")
	domainSetCmd.Flags().BoolVar(&domainSetOn, "on", false, "Enable the selected settings")
	domainSetCmd.Flags().BoolVar(&domainSetOff, "off", false, "Disable the selected settings")
}
