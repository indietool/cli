package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/indietool/cli/domains"
	"github.com/indietool/cli/indietool"
	"github.com/indietool/cli/providers"

	"github.com/spf13/cobra"
)

var (
	domainsRenewOn  bool
	domainsRenewOff bool
)

// findRegistrarForDomain locates the configured registrar that manages the
// given domain by querying each enabled registrar.
func findRegistrarForDomain(ctx context.Context, name string) (domains.Registrar, *domains.ManagedDomain, error) {
	registry := GetProviderRegistry()
	if registry == nil {
		return nil, nil, fmt.Errorf("provider registry not initialized")
	}

	registrars := indietool.GetProviders[domains.Registrar](registry)
	if len(registrars) == 0 {
		return nil, nil, fmt.Errorf("no registrars are configured; run 'indietool config add provider' first")
	}

	for _, reg := range registrars {
		dm, err := reg.GetDomain(ctx, name)
		if err == nil && dm != nil {
			return reg, dm, nil
		}
	}

	return nil, nil, fmt.Errorf("domain %s was not found in any configured registrar", name)
}

// domainsRenewCmd represents the domains renew command
var domainsRenewCmd = &cobra.Command{
	Use:   "renew <domain>",
	Short: "Show renewal info and manage auto-renewal for a domain",
	Long: `Show expiry and auto-renewal status for a domain, or toggle auto-renewal
with --on / --off.

Note: the Cloudflare Registrar API only manages auto-renewal. Renewal pricing
is not exposed by the API, and manual early renewal (paying to extend a domain
before it expires) is only available in the Cloudflare dashboard.

Examples:
  indietool domains renew example.dev
  indietool domains renew example.dev --on
  indietool domains renew example.dev --off
  indietool domains renew example.dev --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if domainsRenewOn && domainsRenewOff {
			return fmt.Errorf("--on and --off are mutually exclusive")
		}

		name := strings.ToLower(strings.TrimSpace(args[0]))
		registrar, dm, err := findRegistrarForDomain(cmd.Context(), name)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if domainsRenewOn || domainsRenewOff {
			enabled := domainsRenewOn
			if err := registrar.UpdateAutoRenewal(cmd.Context(), name, enabled); err != nil {
				return err
			}

			if jsonOutput {
				data, err := json.MarshalIndent(map[string]any{
					"domain":       name,
					"auto_renewal": enabled,
				}, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal result: %w", err)
				}
				fmt.Fprintln(out, string(data))
				return nil
			}

			state := "enabled"
			if !enabled {
				state = "disabled"
			}
			fmt.Fprintf(out, "Auto-renew %s for %s.\n", state, name)
			return nil
		}

		cost, err := registrar.GetRenewalInfo(cmd.Context(), name)
		pricingUnavailable := errors.Is(err, providers.ErrRenewalPricingUnavailable)
		if err != nil && !pricingUnavailable {
			return err
		}

		if jsonOutput {
			payload := map[string]any{
				"domain":       name,
				"provider":     dm.Provider,
				"expiry_date":  dm.ExpiryDate,
				"auto_renewal": dm.AutoRenewal,
			}
			if cost != nil {
				payload["currency"] = cost.Currency
				payload["renewal_cost"] = cost.RenewalPrice
				payload["transfer_cost"] = cost.TransferPrice
			}
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Fprintln(out, string(data))
			return nil
		}

		autoRenew := "off"
		if dm.AutoRenewal {
			autoRenew = "on"
		}
		fmt.Fprintf(out, "Domain:      %s (%s)\n", name, dm.Provider)
		fmt.Fprintf(out, "Expires:     %s\n", dm.ExpiryDate.Format("2006-01-02"))
		fmt.Fprintf(out, "Auto-renew:  %s\n", autoRenew)
		if cost != nil {
			fmt.Fprintf(out, "Renewal:     %.2f %s\n", cost.RenewalPrice, cost.Currency)
		} else {
			fmt.Fprintln(out, "Renewal:     not available via the Registrar API (see the dashboard)")
		}
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Note: the Cloudflare Registrar API only manages auto-renewal; renewal pricing and manual early renewal are dashboard-only.")
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsRenewCmd)

	domainsRenewCmd.Flags().BoolVar(&domainsRenewOn, "on", false, "Enable auto-renewal")
	domainsRenewCmd.Flags().BoolVar(&domainsRenewOff, "off", false, "Disable auto-renewal")
}
