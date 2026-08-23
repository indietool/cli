package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/indietool/cli/domains"
)

const (
	// defaultCloudflareAPIBase is the Cloudflare API v4 base URL.
	defaultCloudflareAPIBase = "https://api.cloudflare.com/client/v4"

	// CloudflareAPIBaseEnvVar overrides the Cloudflare API base URL. It exists
	// for testing and development against mock servers.
	CloudflareAPIBaseEnvVar = "INDIETOOL_CF_API_BASE"

	// maxDomainsPerCheck is the maximum number of domains accepted by the
	// domain-check endpoint per request.
	maxDomainsPerCheck = 20

	// registrationsPerPage is the page size used when listing registrations
	// (the API accepts 1-50).
	registrationsPerPage = 50

	// maxRegistrationPages bounds the cursor-pagination loop so a misbehaving
	// server cannot keep the client paginating forever.
	maxRegistrationPages = 100

	// authScheme is the HTTP authorization scheme used by the Cloudflare API.
	authScheme = "Bearer"
)

// ErrRenewalPricingUnavailable is returned when renewal pricing is requested
// but the current Cloudflare Registrar API does not expose it. The new
// registrations schema carries no price fields; pricing is dashboard-only.
var ErrRenewalPricingUnavailable = errors.New("provider/cloudflare: renewal pricing is not exposed by the Cloudflare Registrar API")

// CloudflareAPIBase returns the Cloudflare API base URL, honoring the
// INDIETOOL_CF_API_BASE override.
func CloudflareAPIBase() string {
	if override := os.Getenv(CloudflareAPIBaseEnvVar); override != "" {
		return strings.TrimRight(override, "/")
	}
	return defaultCloudflareAPIBase
}

// RegistrarPurchaseClient is a thin net/http client for the Cloudflare
// Registrar API (domain-check, registrations, registration-status, plus the
// registration management methods). These endpoints are not part of the
// vendored cloudflare-go v4.5.1 SDK; this client is kept separate so a future
// SDK release can replace it without touching the command layer.
type RegistrarPurchaseClient struct {
	baseURL    string
	accountID  string
	token      string
	sandbox    bool
	httpClient *http.Client

	// PreferAsync forces asynchronous registration behaviour by sending the
	// "Prefer: respond-async" header on registration requests.
	PreferAsync bool
}

// NewRegistrarPurchaseClient creates a purchase client for the given account.
// When sandbox is true, requests target the Registrar Sandbox mirror
// (/accounts/{id}/registrar-sandbox/... on the same host) — a test environment
// where purchases are free but persist.
func NewRegistrarPurchaseClient(accountID, token string, sandbox bool) *RegistrarPurchaseClient {
	return &RegistrarPurchaseClient{
		baseURL:    CloudflareAPIBase(),
		accountID:  accountID,
		token:      token,
		sandbox:    sandbox,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// registrarSegment returns the registrar API path segment for this client.
// The Registrar Sandbox mirrors every production registrar endpoint under
// /registrar-sandbox/ instead of /registrar/.
func (c *RegistrarPurchaseClient) registrarSegment() string {
	if c.sandbox {
		return "registrar-sandbox"
	}
	return "registrar"
}

// registrarEndpoint builds an account-scoped registrar API path, routing
// through the sandbox prefix when sandbox mode is enabled. All registrar
// request paths must be built through this helper.
func (c *RegistrarPurchaseClient) registrarEndpoint(resource string) string {
	return "/accounts/" + c.accountID + "/" + c.registrarSegment() + "/" + resource
}

// apiErrorEntry is a single error entry from the Cloudflare API envelope.
type apiErrorEntry struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// apiResultInfo carries cursor-pagination metadata for list endpoints.
type apiResultInfo struct {
	Count   int    `json:"count"`
	Cursor  string `json:"cursor"`
	PerPage int    `json:"per_page"`
}

// apiEnvelope is the standard Cloudflare API v4 response envelope.
type apiEnvelope struct {
	Success    bool            `json:"success"`
	Errors     []apiErrorEntry `json:"errors"`
	Result     json.RawMessage `json:"result"`
	ResultInfo *apiResultInfo  `json:"result_info,omitempty"`
}

// flexibleFloat parses numbers encoded either as JSON numbers or strings
// (the beta purchase API returns prices as strings, e.g. "10.11").
type flexibleFloat float64

func (f *flexibleFloat) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid numeric value %q: %w", s, err)
	}
	*f = flexibleFloat(v)
	return nil
}

type purchasePricing struct {
	Currency         string        `json:"currency"`
	RegistrationCost flexibleFloat `json:"registration_cost"`
	RenewalCost      flexibleFloat `json:"renewal_cost"`
}

type purchaseDomain struct {
	Name        string          `json:"name"`
	Registrable bool            `json:"registrable"`
	Tier        string          `json:"tier"`
	Reason      string          `json:"reason"`
	Pricing     purchasePricing `json:"pricing"`
}

func (d purchaseDomain) toAvailability() domains.Availability {
	return domains.Availability{
		Name:             d.Name,
		Registrable:      d.Registrable,
		Tier:             d.Tier,
		Reason:           d.Reason,
		Currency:         d.Pricing.Currency,
		RegistrationCost: float64(d.Pricing.RegistrationCost),
		RenewalCost:      float64(d.Pricing.RenewalCost),
	}
}

type registrationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// registrationResource mirrors the Cloudflare Registrar registration object
// (new Registrar API). It doubles as the async workflow-status shape returned
// by registration/update mutations: both share state/completed/error, and the
// resource itself carries the management fields below.
//
// Documented schema (developers.cloudflare.com/api/resources/registrar):
//
//	domain_name, created_at, expires_at, auto_renew, locked,
//	privacy_mode ("redaction"|...), status
//
// Fields absent from that schema (nameservers, renewal price, registrant
// contact) are intentionally NOT modeled — no values are fabricated.
type registrationResource struct {
	DomainName  string             `json:"domain_name"`
	Status      string             `json:"status"`
	State       string             `json:"state"`
	Completed   bool               `json:"completed"`
	Error       *registrationError `json:"error"`
	AutoRenew   bool               `json:"auto_renew"`
	Locked      bool               `json:"locked"`
	PrivacyMode string             `json:"privacy_mode"`
	CreatedAt   *time.Time         `json:"created_at"`
	ExpiresAt   *time.Time         `json:"expires_at"`
}

func (r registrationResource) toResult(fallbackStatus int) *domains.RegistrationResult {
	result := &domains.RegistrationResult{
		DomainName: r.DomainName,
		State:      domains.RegistrationState(r.State),
		Completed:  r.Completed,
	}

	// Derive the state from the HTTP status when the body does not carry it.
	if result.State == "" {
		switch fallbackStatus {
		case http.StatusCreated:
			result.State = domains.RegistrationStateSucceeded
			result.Completed = true
		case http.StatusAccepted:
			result.State = domains.RegistrationStateInProgress
		default:
			result.State = domains.RegistrationStateInProgress
		}
	}

	if r.Error != nil {
		msg := r.Error.Message
		if r.Error.Code != "" {
			msg = fmt.Sprintf("%s (%s)", r.Error.Message, r.Error.Code)
		}
		result.Error = &msg
	}

	return result
}

// toManagedDomain maps a registration resource onto domains.ManagedDomain.
// Fields with no equivalent in the new registrations schema are left at their
// zero value rather than fabricated: Nameservers is empty (nameservers are not
// part of the registration resource) and Cost is nil (the API exposes no
// renewal price). Privacy is derived from privacy_mode ("redaction" means
// WHOIS redaction is on); an absent or unrecognized mode reads as off.
func (r registrationResource) toManagedDomain() domains.ManagedDomain {
	dm := domains.ManagedDomain{
		Name:        r.DomainName,
		Provider:    "cloudflare",
		AutoRenewal: r.AutoRenew,
		Locked:      r.Locked,
		Privacy:     strings.EqualFold(strings.TrimSpace(r.PrivacyMode), "redaction"),
	}
	if r.ExpiresAt != nil {
		dm.ExpiryDate = *r.ExpiresAt
	}
	dm.SetStatus()
	return dm
}

// workflowFailure converts a terminal-but-unsuccessful workflow status into an
// error, mirroring how registration failures surface error.code/message.
func (r registrationResource) workflowFailure(verb string) error {
	switch domains.RegistrationState(r.State) {
	case domains.RegistrationStateFailed, domains.RegistrationStateActionRequired, domains.RegistrationStateBlocked:
	default:
		return nil
	}

	detail := "no details returned"
	if r.Error != nil {
		if r.Error.Code != "" {
			detail = fmt.Sprintf("%s (%s)", r.Error.Message, r.Error.Code)
		} else {
			detail = r.Error.Message
		}
	}
	return fmt.Errorf("provider/cloudflare: %s did not complete (state %s): %s", verb, r.State, detail)
}

// do performs an authenticated request, validates the envelope and decodes
// envelope.Result into out (when non-nil). It returns the HTTP status code.
func (c *RegistrarPurchaseClient) do(ctx context.Context, method, path string, body any, preferAsync bool, out any) (int, error) {
	return c.call(ctx, method, path, body, preferAsync, nil, out)
}

// call is do() with access to the full response envelope. When env is non-nil
// it receives the decoded envelope (including result_info for list endpoints).
func (c *RegistrarPurchaseClient) call(ctx context.Context, method, path string, body any, preferAsync bool, env *apiEnvelope, out any) (int, error) {
	if c.token == "" {
		return 0, fmt.Errorf("provider/cloudflare: the registrar purchase API requires an API token")
	}
	if c.accountID == "" {
		return 0, fmt.Errorf("provider/cloudflare: the registrar purchase API requires an account_id (run 'indietool config add provider cloudflare --account-id')")
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("provider/cloudflare: failed to encode request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, fmt.Errorf("provider/cloudflare: failed to build request: %w", err)
	}
	req.Header.Set("Authorization", authScheme+" "+c.token)
	req.Header.Set("Content-Type", "application/json")
	if preferAsync {
		req.Header.Set("Prefer", "respond-async")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("provider/cloudflare: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("provider/cloudflare: failed to read response: %w", err)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		if resp.StatusCode >= 400 {
			bodySnippet := strings.TrimSpace(string(data))
			if len(bodySnippet) > 512 {
				bodySnippet = bodySnippet[:512] + "...(truncated)"
			}
			return resp.StatusCode, fmt.Errorf("provider/cloudflare: API returned HTTP %d: %s", resp.StatusCode, bodySnippet)
		}
		return resp.StatusCode, fmt.Errorf("provider/cloudflare: failed to parse API response: %w", err)
	}

	if !envelope.Success || resp.StatusCode >= 400 {
		msg := "unknown error"
		if len(envelope.Errors) > 0 {
			parts := make([]string, 0, len(envelope.Errors))
			for _, e := range envelope.Errors {
				parts = append(parts, fmt.Sprintf("%s (code %d)", e.Message, e.Code))
			}
			msg = strings.Join(parts, "; ")
		}
		return resp.StatusCode, fmt.Errorf("provider/cloudflare: API error (HTTP %d): %s", resp.StatusCode, msg)
	}

	if env != nil {
		*env = envelope
	}

	if out != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return resp.StatusCode, fmt.Errorf("provider/cloudflare: failed to parse API result: %w", err)
		}
	}

	return resp.StatusCode, nil
}

// Check performs a real-time availability and pricing check for up to the
// given domain names. Requests are chunked into batches of 20 domains as
// required by the API.
func (c *RegistrarPurchaseClient) Check(ctx context.Context, names []string) ([]domains.Availability, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("provider/cloudflare: no domain names provided")
	}

	results := make([]domains.Availability, 0, len(names))
	for start := 0; start < len(names); start += maxDomainsPerCheck {
		end := start + maxDomainsPerCheck
		if end > len(names) {
			end = len(names)
		}
		chunk := names[start:end]

		var res struct {
			Domains []purchaseDomain `json:"domains"`
		}
		_, err := c.do(
			ctx,
			http.MethodPost,
			c.registrarEndpoint("domain-check"),
			map[string]any{"domains": chunk},
			false,
			&res,
		)
		if err != nil {
			return nil, err
		}

		for _, d := range res.Domains {
			results = append(results, d.toAvailability())
		}
	}

	return results, nil
}

// Register starts a domain registration workflow. This is billable and
// non-refundable once completed. The response is either 201 Created
// (completed) or 202 Accepted (still in progress; poll RegistrationStatus).
func (c *RegistrarPurchaseClient) Register(ctx context.Context, name string) (*domains.RegistrationResult, error) {
	var res registrationResource
	status, err := c.do(
		ctx,
		http.MethodPost,
		c.registrarEndpoint("registrations"),
		map[string]any{"domain_name": name},
		c.PreferAsync,
		&res,
	)
	if err != nil {
		return nil, err
	}

	if res.DomainName == "" {
		res.DomainName = name
	}
	return res.toResult(status), nil
}

// RegistrationStatus polls the registration workflow state for a domain.
func (c *RegistrarPurchaseClient) RegistrationStatus(ctx context.Context, name string) (*domains.RegistrationResult, error) {
	var res registrationResource
	status, err := c.do(
		ctx,
		http.MethodGet,
		c.registrarEndpoint("registrations/"+url.PathEscape(name)+"/registration-status"),
		nil,
		false,
		&res,
	)
	if err != nil {
		return nil, err
	}

	if res.DomainName == "" {
		res.DomainName = name
	}
	return res.toResult(status), nil
}

// GetRegistration fetches a single registration resource via the new
// Registrar API (GET /accounts/{account}/registrar/registrations/{domain}) and
// maps it onto domains.ManagedDomain.
func (c *RegistrarPurchaseClient) GetRegistration(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	var res registrationResource
	if _, err := c.do(
		ctx,
		http.MethodGet,
		c.registrarEndpoint("registrations/"+url.PathEscape(name)),
		nil,
		false,
		&res,
	); err != nil {
		return nil, err
	}

	dm := res.toManagedDomain()
	if dm.Name == "" {
		dm.Name = name
	}
	return &dm, nil
}

// UpdateAutoRenew toggles auto-renewal for a registration (PATCH
// /accounts/{account}/registrar/registrations/{domain}, body {"auto_renew":
// bool}). This is currently the only mutation the new Registrar API supports;
// registrar lock and WHOIS privacy have no API equivalent yet. The endpoint
// may answer with an async workflow status; terminal failure states are
// surfaced as errors, while pending/in_progress are treated as accepted.
func (c *RegistrarPurchaseClient) UpdateAutoRenew(ctx context.Context, name string, enabled bool) error {
	var res registrationResource
	if _, err := c.do(
		ctx,
		http.MethodPatch,
		c.registrarEndpoint("registrations/"+url.PathEscape(name)),
		map[string]any{"auto_renew": enabled},
		false,
		&res,
	); err != nil {
		return err
	}

	if failErr := res.workflowFailure("auto-renew update"); failErr != nil {
		return fmt.Errorf("provider/cloudflare: auto-renew update for %s: %w", name, failErr)
	}
	return nil
}

// ListRegistrations lists all registrations on the account via the new
// Registrar API (GET /accounts/{account}/registrar/registrations), following
// cursor pagination until exhausted.
func (c *RegistrarPurchaseClient) ListRegistrations(ctx context.Context) ([]domains.ManagedDomain, error) {
	results := make([]domains.ManagedDomain, 0)
	cursor := ""

	for page := 0; ; page++ {
		if page >= maxRegistrationPages {
			return nil, fmt.Errorf("provider/cloudflare: listing registrations exceeded %d pages; aborting", maxRegistrationPages)
		}

		query := url.Values{}
		query.Set("per_page", strconv.Itoa(registrationsPerPage))
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		var env apiEnvelope
		var registrations []registrationResource
		if _, err := c.call(
			ctx,
			http.MethodGet,
			c.registrarEndpoint("registrations")+"?"+query.Encode(),
			nil,
			false,
			&env,
			&registrations,
		); err != nil {
			return nil, err
		}

		for _, reg := range registrations {
			results = append(results, reg.toManagedDomain())
		}

		cursor = ""
		if env.ResultInfo != nil {
			cursor = env.ResultInfo.Cursor
		}
		if cursor == "" {
			return results, nil
		}
	}
}
