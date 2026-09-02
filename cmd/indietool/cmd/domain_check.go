package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/indietool/cli/domains"
	"github.com/indietool/cli/output"

	"github.com/spf13/cobra"
)

// getCloudflarePurchaser returns the Purchaser capability of the configured
// Cloudflare provider, failing fast with prerequisite guidance when it is
// missing.
func getCloudflarePurchaser() (domains.Purchaser, error) {
	cfg := GetConfig()
	if cfg == nil || cfg.Providers.Cloudflare == nil || !cfg.Providers.Cloudflare.Enabled {
		return nil, fmt.Errorf("cloudflare provider is not configured; run 'indietool config add provider cloudflare' with the --api-token and --account-id flags")
	}
	if cfg.Providers.Cloudflare.AccountId == "" {
		return nil, fmt.Errorf("cloudflare account_id is missing; run 'indietool config add provider cloudflare' with the --api-token and --account-id flags")
	}

	registry := GetProviderRegistry()
	if registry == nil {
		return nil, fmt.Errorf("provider registry not initialized")
	}

	provider, ok := registry.Get("cloudflare")
	if !ok {
		return nil, fmt.Errorf("cloudflare provider not found in registry")
	}

	purchaser, ok := domains.AsPurchaser(provider.AsRegistrar())
	if !ok {
		return nil, fmt.Errorf("cloudflare provider does not support domain purchasing")
	}
	return purchaser, nil
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

// domainCheckCmd represents the domain check command
var domainCheckCmd = &cobra.Command{
	Use:   "check <domain>...",
	Short: "Check real-time domain availability and pricing (Cloudflare Registrar beta)",
	Long: `Check real-time availability and pricing for one or more domains through the
Cloudflare Registrar purchase API (beta).

The check queries the registry directly and reflects current availability.
Up to 20 domains can be checked per invocation.

Examples:
  indietool domain check example.dev
  indietool domain check example.com example.dev example.app
  indietool domain check example.dev --json`,
	Args: cobra.RangeArgs(1, 20),
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := normalizeDomainNames(args)
		if err != nil {
			return err
		}

		purchaser, err := getCloudflarePurchaser()
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
}
