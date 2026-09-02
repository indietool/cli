package domains

import "context"

// Availability describes the availability and pricing of a single domain as
// reported by a registrar's availability/check endpoint.
type Availability struct {
	Name             string  `json:"name"`
	Registrable      bool    `json:"registrable"`
	Tier             string  `json:"tier,omitempty"`
	Reason           string  `json:"reason,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	RegistrationCost float64 `json:"registration_cost,omitempty"`
	RenewalCost      float64 `json:"renewal_cost,omitempty"`
}

// RegistrationState is the workflow state of a domain registration.
type RegistrationState string

const (
	RegistrationStateInProgress     RegistrationState = "in_progress"
	RegistrationStateSucceeded      RegistrationState = "succeeded"
	RegistrationStateFailed         RegistrationState = "failed"
	RegistrationStateActionRequired RegistrationState = "action_required"
	RegistrationStateBlocked        RegistrationState = "blocked"
)

// RegistrationResult is the outcome of a domain registration workflow.
type RegistrationResult struct {
	DomainName string            `json:"domain_name"`
	State      RegistrationState `json:"state"` // in_progress|succeeded|failed|action_required|blocked
	Completed  bool              `json:"completed"`
	Error      *string           `json:"error,omitempty"`
}

// IsTerminal reports whether the registration workflow has reached a final
// state and no further polling is needed.
func (r *RegistrationResult) IsTerminal() bool {
	switch r.State {
	case RegistrationStateSucceeded,
		RegistrationStateFailed,
		RegistrationStateActionRequired,
		RegistrationStateBlocked:
		return true
	default:
		return false
	}
}

// PostalAddress is the registrant's postal address as required by registrar
// registration payloads. CountryCode is ISO 3166-1 alpha-2.
type PostalAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
}

// PostalInfo is the registrant's named postal record.
type PostalInfo struct {
	Name         string         `json:"name,omitempty"`
	Organization string         `json:"organization,omitempty"`
	Address      *PostalAddress `json:"address,omitempty"`
}

// RegistrantContact carries the registrant contact data required by
// registrars without an account-level address book (e.g. the Cloudflare
// Registrar Sandbox). Phone uses the "+<cc>.<number>" format (e.g.
// "+1.5555551234"). JSON tags mirror the Cloudflare registration schema.
type RegistrantContact struct {
	Phone      string      `json:"phone,omitempty"`
	Email      string      `json:"email,omitempty"`
	Fax        string      `json:"fax,omitempty"`
	PostalInfo *PostalInfo `json:"postal_info,omitempty"`
}

// Purchaser is an optional capability for registrars that support buying
// domains (real-time availability checks and registration). Use AsPurchaser
// to type-assert a Registrar.
type Purchaser interface {
	Check(ctx context.Context, names []string) ([]Availability, error)
	Register(ctx context.Context, name string, contact *RegistrantContact) (*RegistrationResult, error)
	RegistrationStatus(ctx context.Context, name string) (*RegistrationResult, error)
}

// AsPurchaser returns the Purchaser capability of a Registrar if it is
// supported.
func AsPurchaser(r Registrar) (Purchaser, bool) {
	p, ok := r.(Purchaser)
	return p, ok
}
