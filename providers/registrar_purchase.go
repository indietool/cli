package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// authScheme is the HTTP authorization scheme used by the Cloudflare API.
	authScheme = "Bearer"
)

// CloudflareAPIBase returns the Cloudflare API base URL, honoring the
// INDIETOOL_CF_API_BASE override.
func CloudflareAPIBase() string {
	if override := os.Getenv(CloudflareAPIBaseEnvVar); override != "" {
		return strings.TrimRight(override, "/")
	}
	return defaultCloudflareAPIBase
}

// RegistrarPurchaseClient is a thin net/http client for the beta Cloudflare
// Registrar purchase API (domain-check, registrations, registration-status).
// These endpoints are not part of the vendored cloudflare-go v4.5.1 SDK; this
// client is kept separate so a future SDK release can replace it without
// touching the command layer.
type RegistrarPurchaseClient struct {
	baseURL    string
	accountID  string
	token      string
	httpClient *http.Client

	// PreferAsync forces asynchronous registration behaviour by sending the
	// "Prefer: respond-async" header on registration requests.
	PreferAsync bool
}

// NewRegistrarPurchaseClient creates a purchase client for the given account.
func NewRegistrarPurchaseClient(accountID, token string) *RegistrarPurchaseClient {
	return &RegistrarPurchaseClient{
		baseURL:    CloudflareAPIBase(),
		accountID:  accountID,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// apiErrorEntry is a single error entry from the Cloudflare API envelope.
type apiErrorEntry struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// apiEnvelope is the standard Cloudflare API v4 response envelope.
type apiEnvelope struct {
	Success bool            `json:"success"`
	Errors  []apiErrorEntry    `json:"errors"`
	Result  json.RawMessage `json:"result"`
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

type registrationResource struct {
	DomainName string             `json:"domain_name"`
	State      string             `json:"state"`
	Completed  bool               `json:"completed"`
	Error      *registrationError `json:"error"`
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

// do performs an authenticated request and decodes the result into out.
func (c *RegistrarPurchaseClient) do(ctx context.Context, method, path string, body any, preferAsync bool, out any) (int, error) {
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
			return resp.StatusCode, fmt.Errorf("provider/cloudflare: API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
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
			fmt.Sprintf("/accounts/%s/registrar/domain-check", c.accountID),
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
		fmt.Sprintf("/accounts/%s/registrar/registrations", c.accountID),
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
		fmt.Sprintf("/accounts/%s/registrar/registrations/%s/registration-status", c.accountID, name),
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
