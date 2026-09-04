package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/indietool/cli/domains"

	"github.com/spf13/cobra"
)

var (
	domainRegisterYes      bool
	domainRegisterDryRun   bool
	domainRegisterProvider string

	domainRegisterContactName    string
	domainRegisterContactOrg     string
	domainRegisterContactEmail   string
	domainRegisterContactPhone   string
	domainRegisterContactStreet  string
	domainRegisterContactCity    string
	domainRegisterContactState   string
	domainRegisterContactPostal  string
	domainRegisterContactCountry string

	// registerPollInterval and registerPollTimeout control the registration
	// status polling loop. They are variables so tests can shorten them.
	registerPollInterval = 3 * time.Second
	registerPollTimeout  = 60 * time.Second
)

// confirmPrompt asks the user a yes/no question on the command's input.
func confirmPrompt(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", prompt)

	var answer string
	if _, err := fmt.Fscanln(cmd.InOrStdin(), &answer); err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// awaitRegistration polls the registration workflow until it reaches a
// terminal state or the poll timeout is exceeded.
func awaitRegistration(cmd *cobra.Command, purchaser domains.Purchaser, name string, result *domains.RegistrationResult) (*domains.RegistrationResult, error) {
	if result.IsTerminal() {
		return result, nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Registration in progress, polling for completion...\n")

	deadline := time.Now().Add(registerPollTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(registerPollInterval)

		status, err := purchaser.RegistrationStatus(cmd.Context(), name)
		if err != nil {
			return nil, fmt.Errorf("failed to poll registration status: %w", err)
		}
		result = status
		if result.IsTerminal() {
			return result, nil
		}
	}

	return result, fmt.Errorf("registration of %s is still in progress after %s; check its status in the Cloudflare dashboard", name, registerPollTimeout)
}

// registrationError converts a non-success terminal registration state into
// an actionable error.
func registrationError(name string, result *domains.RegistrationResult) error {
	detail := ""
	if result.Error != nil && *result.Error != "" {
		detail = ": " + *result.Error
	}

	switch result.State {
	case domains.RegistrationStateActionRequired:
		base := "complete the required action in the Cloudflare dashboard (Domains > Registrations), then check the domain status"
		if detail != "" {
			base = strings.TrimPrefix(detail, ": ")
		}
		return fmt.Errorf("registration of %s requires action: %s", name, base)
	case domains.RegistrationStateBlocked:
		return fmt.Errorf("registration of %s is blocked%s; contact Cloudflare support or check the dashboard", name, detail)
	default:
		return fmt.Errorf("registration of %s failed%s", name, detail)
	}
}

// buildRegistrantContact assembles the optional registrant contact from the
// --contact-* flags. With no contact flag set it returns nil so the API falls
// back to the account's default address book (or Express registration). When
// any contact flag is set, every field required by the Cloudflare registration
// schema must be present; the error names each missing flag.
func buildRegistrantContact() (*domains.RegistrantContact, error) {
	fields := []struct {
		flag  string
		value string
	}{
		{"--contact-name", domainRegisterContactName},
		{"--contact-email", domainRegisterContactEmail},
		{"--contact-phone", domainRegisterContactPhone},
		{"--contact-street", domainRegisterContactStreet},
		{"--contact-city", domainRegisterContactCity},
		{"--contact-state", domainRegisterContactState},
		{"--contact-postal-code", domainRegisterContactPostal},
		{"--contact-country", domainRegisterContactCountry},
	}

	anySet := false
	for _, f := range fields {
		if strings.TrimSpace(f.value) != "" {
			anySet = true
			break
		}
	}
	if !anySet {
		return nil, nil
	}

	var missing []string
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			missing = append(missing, f.flag)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("incomplete registrant contact: missing %s (or clear all --contact-* flags to use the account's default contact)", strings.Join(missing, ", "))
	}

	country := strings.ToUpper(strings.TrimSpace(domainRegisterContactCountry))
	if len(country) != 2 {
		return nil, fmt.Errorf("--contact-country must be an ISO 3166-1 alpha-2 code (e.g. US), got %q", domainRegisterContactCountry)
	}

	return &domains.RegistrantContact{
		Phone: strings.TrimSpace(domainRegisterContactPhone),
		Email: strings.TrimSpace(domainRegisterContactEmail),
		PostalInfo: &domains.PostalInfo{
			Name:         strings.TrimSpace(domainRegisterContactName),
			Organization: strings.TrimSpace(domainRegisterContactOrg),
			Address: &domains.PostalAddress{
				Street:      strings.TrimSpace(domainRegisterContactStreet),
				City:        strings.TrimSpace(domainRegisterContactCity),
				State:       strings.TrimSpace(domainRegisterContactState),
				PostalCode:  strings.TrimSpace(domainRegisterContactPostal),
				CountryCode: country,
			},
		},
	}, nil
}

// domainRegisterCmd represents the domain register command
var domainRegisterCmd = &cobra.Command{
	Use:   "register <domain>",
	Short: "Register a domain via a purchase-capable registrar (billable)",
	Long: `Register a domain through a purchase-capable registrar API.

This is a billable, non-refundable operation. The current price is shown and
must be confirmed (or acknowledged with --yes) before the registration is
submitted.

Providers (choose with --provider, or the first configured provider is used):
  cloudflare  Cloudflare Registrar (beta). Registrations may complete
              asynchronously; the command polls until terminal state.
  porkbun     Porkbun API v3. Registration is synchronous and paid from
              account credit: the account needs email+phone verified,
              sufficient credit, and at least one prior registration
              (Porkbun requirement). Premium domains cannot be registered
              via the API. Use pk1_sb_/sk1_sb_ sandbox keys to test without
              real charges (same base URL; fake credit).

Registrant contact (--contact-* flags, Cloudflare only):
  The Cloudflare Registrar Sandbox requires full registrant contact data on
  every registration; production Express-mode accounts do not. Pass all
  --contact-* flags together — --contact-country is ISO 3166-1 alpha-2 and
  --contact-phone uses the "+1.5555551234" format. Without contact flags the
  account's default address book entry is used. Porkbun does not accept
  registrant contacts via the API: it registers with the account's default
  contact, so --contact-* flags are refused for --provider porkbun.

Examples:
  indietool domain register example.dev
  indietool domain register example.dev --yes
  indietool domain register example.dev --dry-run
  indietool domain register example.com --provider porkbun --yes
  indietool domain register example.dev --yes \
    --contact-name "Jane Doe" --contact-email jane@example.com \
    --contact-phone "+1.5555551234" --contact-street "1 Main St" \
    --contact-city "Springfield" --contact-state CA \
    --contact-postal-code "90210" --contact-country US`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := normalizeDomainNames(args)
		if err != nil {
			return err
		}
		name := names[0]

		contact, err := buildRegistrantContact()
		if err != nil {
			return err
		}

		purchaser, err := getPurchaser(domainRegisterProvider)
		if err != nil {
			return err
		}

		// Porkbun registers with the account's default registrant contact;
		// fail fast before showing prices instead of after confirmation.
		if contact != nil && purchaserProviderName(purchaser) == "porkbun" {
			return fmt.Errorf("--contact-* flags are not supported for Porkbun: registrations use the account's default registrant contact (manage it at porkbun.com/account)")
		}

		// Real-time availability + price immediately before registration.
		availability, err := purchaser.Check(cmd.Context(), []string{name})
		if err != nil {
			return err
		}
		if len(availability) == 0 {
			return fmt.Errorf("no availability result returned for %s", name)
		}
		avail := availability[0]
		if !avail.Registrable {
			reason := avail.Reason
			if reason == "" {
				reason = "domain unavailable"
			}
			return fmt.Errorf("%s is not registrable via the Cloudflare Registrar API: %s", name, reason)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Domain: %s\n", avail.Name)
		if avail.Currency != "" {
			fmt.Fprintf(out, "Price:  %.2f %s registration, %.2f %s renewal\n",
				avail.RegistrationCost, avail.Currency, avail.RenewalCost, avail.Currency)
		} else {
			fmt.Fprintf(out, "Price:  %.2f registration, %.2f renewal\n",
				avail.RegistrationCost, avail.RenewalCost)
		}

		if domainRegisterDryRun {
			fmt.Fprintln(out, "Dry run: no registration performed.")
			return nil
		}

		if !domainRegisterYes {
			fmt.Fprintln(out, "Registration is billable and non-refundable.")
			confirmed, err := confirmPrompt(cmd, fmt.Sprintf("Register %s?", name))
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			if !confirmed {
				return fmt.Errorf("registration of %s aborted", name)
			}
		}

		result, err := purchaser.Register(cmd.Context(), name, contact)
		if err != nil {
			return err
		}

		result, err = awaitRegistration(cmd, purchaser, name, result)
		if err != nil {
			return err
		}

		if jsonOutput {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal registration result: %w", err)
			}
			fmt.Fprintln(out, string(data))
		}

		if result.State != domains.RegistrationStateSucceeded {
			return registrationError(name, result)
		}

		if !jsonOutput {
			fmt.Fprintf(out, "Registration of %s succeeded.\n", name)
			if purchaserProviderName(purchaser) == "porkbun" {
				fmt.Fprintf(out, "Manage auto-renew with 'indietool domains renew %s --on|--off'.\n", name)
			} else {
				fmt.Fprintf(out, "Note: auto-renew defaults to off for API registrations; enable it with 'indietool domains renew %s --on'.\n", name)
			}
		}
		return nil
	},
}

func init() {
	domainCmd.AddCommand(domainRegisterCmd)

	domainRegisterCmd.Flags().BoolVar(&domainRegisterYes, "yes", false, "Skip the interactive price confirmation (billable)")
	domainRegisterCmd.Flags().BoolVar(&domainRegisterDryRun, "dry-run", false, "Show availability and price without registering")
	domainRegisterCmd.Flags().StringVar(&domainRegisterProvider, "provider", "", "Purchase provider: cloudflare or porkbun (default: first configured provider)")

	domainRegisterCmd.Flags().StringVar(&domainRegisterContactName, "contact-name", "", "Registrant full name (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactOrg, "contact-organization", "", "Registrant organization (optional)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactEmail, "contact-email", "", "Registrant email (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactPhone, "contact-phone", "", "Registrant phone in \"+1.5555551234\" format (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactStreet, "contact-street", "", "Registrant street address (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactCity, "contact-city", "", "Registrant city (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactState, "contact-state", "", "Registrant state/province (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactPostal, "contact-postal-code", "", "Registrant postal code (required with any --contact-* flag)")
	domainRegisterCmd.Flags().StringVar(&domainRegisterContactCountry, "contact-country", "", "Registrant ISO 3166-1 alpha-2 country code, e.g. US (required with any --contact-* flag)")
}
