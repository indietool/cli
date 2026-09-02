package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/indietool/cli/domains"
	"github.com/indietool/cli/indietool"
	"github.com/indietool/cli/output"

	"github.com/spf13/cobra"
)

// purchaserProviderName returns the provider name backing a Purchaser, or
// "" when it cannot be determined.
func purchaserProviderName(p domains.Purchaser) string {
	if prov, ok := p.(indietool.Provider); ok {
		return prov.Name()
	}
	return ""
}

// validatePurchaser applies per-provider prerequisite checks so
// misconfigurations fail fast with actionable guidance before any API call.
func validatePurchaser(p domains.Purchaser) (domains.Purchaser, error) {
	if purchaserProviderName(p) == "cloudflare" {
		cfg := GetConfig()
		if cfg == nil || cfg.Providers.Cloudflare == nil || !cfg.Providers.Cloudflare.Enabled {
			return nil, fmt.Errorf("cloudflare provider is not configured; run 'indietool config add provider cloudflare' with the --api-token and --account-id flags")
		}
		if cfg.Providers.Cloudflare.AccountId == "" {
			return nil, fmt.Errorf("cloudflare account_id is missing; run 'indietool config add provider cloudflare' with the --api-token and --account-id flags")
		}
	}
	return p, nil
}

// getPurchaser returns the Purchaser capability of the named provider, or of
// the first enabled purchase-capable provider when name is empty. Supported
// purchase providers are cloudflare and porkbun.
func getPurchaser(name string) (domains.Purchaser, error) {
	registry := GetProviderRegistry()
	if registry == nil {
		return nil, fmt.Errorf("provider registry not initialized")
	}

	if name != "" {
		provider, ok := registry.Get(strings.ToLower(name))
		if !ok {
			return nil, fmt.Errorf("provider %q is not configured; run 'indietool config add provider %s' first", name, name)
		}
		purchaser, ok := domains.AsPurchaser(provider.AsRegistrar())
		if !ok {
			return nil, fmt.Errorf("provider %q does not support domain purchasing", name)
		}
		return validatePurchaser(purchaser)
	}

	candidates := indietool.GetProviders[domains.Purchaser](registry)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no purchase-capable providers are configured; run 'indietool config add provider cloudflare' or 'indietool config add provider porkbun' first")
	}

	// Prefer the first candidate whose prerequisites check out; keep the
	// first validation error around for reporting when none do.
	var firstErr error
	for _, candidate := range candidates {
		validated, err := validatePurchaser(candidate)
		if err == nil {
			return validated, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, firstErr
}

// normalizeDomainNames lower-cases and validates the given domain names.
func normalizeDomainNames(args []string) ([]string, error) {
	names := make([]string, 0, len(args))
	for _, arg := range args {
		name := strings.ToLower(strings.TrimSpace(arg))
		if name == "" {
			continue
		}
		if !strings.Contains(name, ".") {
			return nil, fmt.Errorf("%q is not a full domain name (include the TLD, e.g. example.com)", arg)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no domain names provided")
	}
	return names, nil
}

// priceFormatter renders a price value, using "-" for absent prices.
func priceFormatter(value interface{}) string {
	switch v := value.(type) {
	case float64:
		if v == 0 {
			return "-"
		}
		return fmt.Sprintf("%.2f", v)
	case nil:
		return "-"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// availabilityTableConfig renders domains.Availability rows.
var availabilityTableConfig = output.TableConfig{
	DefaultColumns: []output.Column{
		{Name: "NAME", JSONPath: "name", Required: true},
		{Name: "AVAILABLE", JSONPath: "registrable", Formatter: output.YesNoFormatter, Required: true},
		{Name: "PRICE", JSONPath: "registration_cost", Formatter: priceFormatter, Required: true},
		{Name: "RENEWAL", JSONPath: "renewal_cost", Formatter: priceFormatter, Required: true},
		{Name: "CURRENCY", JSONPath: "currency"},
		{Name: "TIER", JSONPath: "tier"},
		{Name: "REASON", JSONPath: "reason"},
	},
}

// domainCheckProvider selects the purchase provider for checks
// (cloudflare or porkbun; empty = auto-detect).
var domainCheckProvider string

// domainCheckCmd represents the domain check command
var domainCheckCmd = &cobra.Command{
	Use:   "check <domain>...",
	Short: "Check real-time domain availability and pricing (Cloudflare or Porkbun)",
	Long: `Check real-time availability and pricing for one or more domains through a
purchase-capable registrar API.

Providers (choose with --provider, or the first configured provider is used):
  cloudflare  Cloudflare Registrar (beta), up to 20 domains per request
  porkbun     Porkbun API v3, one domain per request; checks are paced to the
              account's rate-limit window (default 1 check per 10 seconds),
              so large batches take a while. Premium domains are reported as
              not registrable (the Porkbun API cannot register them).

The check queries the registry directly and reflects current availability.
Up to 20 domains can be checked per invocation.

Examples:
  indietool domain check example.dev
  indietool domain check example.com example.dev example.app
  indietool domain check example.dev --json
  indietool domain check example.com --provider porkbun`,
	Args: cobra.RangeArgs(1, 20),
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := normalizeDomainNames(args)
		if err != nil {
			return err
		}

		purchaser, err := getPurchaser(domainCheckProvider)
		if err != nil {
			return err
		}

		availability, err := purchaser.Check(cmd.Context(), names)
		if err != nil {
			return err
		}

		if jsonOutput {
			data, err := json.MarshalIndent(availability, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal availability results: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		table := output.NewTable(availabilityTableConfig, output.TableOptions{
			Format:  output.FormatTable,
			NoColor: true,
			Writer:  cmd.OutOrStdout(),
		})
		table.AddRows(availability)
		return table.Render()
	},
}

func init() {
	domainCmd.AddCommand(domainCheckCmd)

	domainCheckCmd.Flags().StringVar(&domainCheckProvider, "provider", "", "Purchase provider: cloudflare or porkbun (default: first configured provider)")
}
