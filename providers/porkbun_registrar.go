package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/indietool/cli/domains"
)

const (
	// defaultPorkbunAPIBase is the Porkbun API v3 base URL. The documented
	// endpoints live under api.porkbun.com (not porkbun.com).
	defaultPorkbunAPIBase = "https://api.porkbun.com/api/json/v3"

	// PorkbunAPIBaseEnvVar overrides the Porkbun API base URL. It exists for
	// testing and development against mock servers.
	PorkbunAPIBaseEnvVar = "INDIETOOL_PORKBUN_API_BASE"

	// porkbunDefaultRetryWait is the fallback pause on a 429 that carries
	// no (or an unparseable) Retry-After header.
	porkbunDefaultRetryWait = 1 * time.Second

	// porkbunMaxRetryWait caps how long a 429 Retry-After pause may block.
	porkbunMaxRetryWait = 60 * time.Second

	// porkbunPacingGrace is added to a reported rate-limit window so the
	// next request lands after the window resets, not exactly on it.
	porkbunPacingGrace = 1 * time.Second

	// porkbunMaxPacingWait caps a single pacing pause between checks.
	porkbunMaxPacingWait = 15 * time.Second
)

// PorkbunAPIBase returns the Porkbun API base URL, honoring the
// INDIETOOL_PORKBUN_API_BASE override.
func PorkbunAPIBase() string {
	if override := os.Getenv(PorkbunAPIBaseEnvVar); override != "" {
		return strings.TrimRight(override, "/")
	}
	return defaultPorkbunAPIBase
}

// PorkbunRegistrarClient is a thin net/http client for the Porkbun API v3
// registrar endpoints (domain availability, registration, auto-renewal, and
// domain detail). The vendored tuzzmaniandevil/porkbun-go SDK covers DNS,
// nameservers, and domain listing but none of the purchase endpoints; this
// client is kept separate so a future SDK release can replace it without
// touching the command layer.
//
// Authentication uses the API's header scheme (X-API-Key / X-Secret-API-Key).
// Sandbox accounts (keys prefixed pk1_sb_ / sk1_sb_) hit the same base URL —
// no special routing is required.
type PorkbunRegistrarClient struct {
	baseURL    string
	apiKey     string
	secretKey  string
	httpClient *http.Client
}

// NewPorkbunRegistrarClient creates a registrar client for the given
// credentials.
func NewPorkbunRegistrarClient(apiKey, secretKey string) *PorkbunRegistrarClient {
	return &PorkbunRegistrarClient{
		baseURL:    PorkbunAPIBase(),
		apiKey:     apiKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// porkbunNextAction is the API's machine-readable remediation hint attached
// to many error responses.
type porkbunNextAction struct {
	Type string `json:"type,omitempty"`
	Hint string `json:"hint,omitempty"`
	URL  string `json:"url,omitempty"`
}

// porkbunResponse is the common envelope of every Porkbun API response:
// status is "SUCCESS" or "ERROR" (with message/code on errors). The typed
// response structs embed it so a single unmarshal covers envelope + payload.
type porkbunResponse struct {
	Status     string             `json:"status"`
	Message    string             `json:"message,omitempty"`
	Code       string             `json:"code,omitempty"`
	NextAction *porkbunNextAction `json:"next_action,omitempty"`
	RequestID  string             `json:"requestId,omitempty"`
}

// flexibleInt parses a JSON field that may arrive as a number or a numeric
// string. The domain/get payload mixes both in practice (live observation:
// securityLock: 0 but autoRenew: "1", whoisPrivacy: "1").
type flexibleInt int

func (f *flexibleInt) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid integer value %q: %w", s, err)
	}
	*f = flexibleInt(v)
	return nil
}

// porkbunPriceInfo is a price entry (registration, renewal, or transfer).
// Prices are strings in USD dollars ("9.73").
type porkbunPriceInfo struct {
	Type         string `json:"type,omitempty"`
	Price        string `json:"price,omitempty"`
	RegularPrice string `json:"regularPrice,omitempty"`
}

// porkbunCheckResponse is the checkDomain/{domain} response. One domain per
// request. TTLRemaining carries the seconds until the account's check rate
// limit window resets (0 when not rate limited).
type porkbunCheckResponse struct {
	porkbunResponse
	Response struct {
		Avail          string `json:"avail"`
		Type           string `json:"type,omitempty"`
		Price          string `json:"price,omitempty"`
		FirstYearPromo string `json:"firstYearPromo,omitempty"`
		RegularPrice   string `json:"regularPrice,omitempty"`
		Premium        string `json:"premium,omitempty"`
		MinDuration    int    `json:"minDuration,omitempty"`
		Additional     struct {
			Renewal  *porkbunPriceInfo `json:"renewal,omitempty"`
			Transfer *porkbunPriceInfo `json:"transfer,omitempty"`
		} `json:"additional,omitempty"`
	} `json:"response"`
	TTLRemaining int `json:"ttlRemaining"`
}

// toAvailability maps a checkDomain response onto domains.Availability.
// Premium names are reported as not registrable: the Porkbun API refuses to
// register them, so presenting them as purchasable would be misleading.
func (r *porkbunCheckResponse) toAvailability(name string) domains.Availability {
	avail := domains.Availability{
		Name:     name,
		Currency: "USD",
	}

	premium := strings.EqualFold(r.Response.Premium, "yes")
	switch {
	case strings.EqualFold(r.Response.Avail, "yes") && premium:
		avail.Tier = "premium"
		avail.Registrable = false
		avail.Reason = "premium domains cannot be registered via the Porkbun API"
	case strings.EqualFold(r.Response.Avail, "yes"):
		avail.Registrable = true
		avail.RegistrationCost = parsePorkbunPrice(r.Response.Price)
		if r.Response.Additional.Renewal != nil {
			avail.RenewalCost = parsePorkbunPrice(r.Response.Additional.Renewal.Price)
		}
		if strings.EqualFold(r.Response.FirstYearPromo, "yes") {
			avail.Tier = "first-year-promo"
		}
	default:
		avail.Registrable = false
		avail.Reason = "unavailable"
	}

	return avail
}

// porkbunCreateRequest is the create/{domain} (registration) payload. Cost
// is integer pennies and must exactly equal the current registration price
// at the domain's minimum duration, as quoted by checkDomain.
type porkbunCreateRequest struct {
	Cost         int64  `json:"cost"`
	AgreeToTerms string `json:"agreeToTerms"`
	DryRun       bool   `json:"dryRun,omitempty"`
	WhoisPrivacy *bool  `json:"whoisPrivacy,omitempty"`
}

// porkbunCreateResponse is the create/{domain} response. Registrations are
// synchronous: a SUCCESS response carries the orderId and the charged cost.
// When the request sets dryRun the API returns a preview instead (DryRun
// true, WouldSucceed reporting the outcome) without creating an order.
type porkbunCreateResponse struct {
	porkbunResponse
	Domain          string `json:"domain,omitempty"`
	Cost            int64  `json:"cost,omitempty"`
	OrderID         int64  `json:"orderId,omitempty"`
	Balance         int64  `json:"balance,omitempty"`
	DryRun          bool   `json:"dryRun,omitempty"`
	WouldSucceed    bool   `json:"wouldSucceed,omitempty"`
	CostDisplay     string `json:"costDisplay,omitempty"`
	SufficientFunds bool   `json:"sufficientFunds,omitempty"`
}

// porkbunUpdateAutoRenewRequest toggles auto-renewal for a domain.
type porkbunUpdateAutoRenewRequest struct {
	Status string `json:"status"` // "on" | "off"
}

// porkbunGetResponse is the GET domain/get/{domain} response.
type porkbunGetResponse struct {
	porkbunResponse
	Domain struct {
		Domain       string      `json:"domain"`
		Status       string      `json:"status,omitempty"`
		TLD          string      `json:"tld,omitempty"`
		CreateDate   string      `json:"createDate,omitempty"`
		ExpireDate   string      `json:"expireDate,omitempty"`
		SecurityLock flexibleInt `json:"securityLock,omitempty"`
		WhoisPrivacy flexibleInt `json:"whoisPrivacy,omitempty"`
		AutoRenew    flexibleInt `json:"autoRenew,omitempty"`
		APIAccess    flexibleInt `json:"apiAccess,omitempty"`
		NotLocal     flexibleInt `json:"notLocal,omitempty"`
	} `json:"domain"`
}

// toManagedDomain maps a domain/get payload onto domains.ManagedDomain.
// Nameservers are not part of the payload and are left empty rather than
// issuing an extra API call.
func (r *porkbunGetResponse) toManagedDomain() *domains.ManagedDomain {
	dm := &domains.ManagedDomain{
		Name:        r.Domain.Domain,
		Provider:    "porkbun",
		AutoRenewal: r.Domain.AutoRenew == 1,
		Locked:      r.Domain.SecurityLock == 1,
		Privacy:     r.Domain.WhoisPrivacy == 1,
		LastUpdated: time.Now(),
	}
	if t, err := parsePorkbunTime(r.Domain.ExpireDate); err == nil {
		dm.ExpiryDate = t
	}
	dm.SetStatus()
	return dm
}

// parsePorkbunPrice converts a Porkbun price string ("9.73") into dollars.
func parsePorkbunPrice(price string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(price), 64)
	if err != nil {
		return 0
	}
	return v
}

// priceToPennies converts a price string ("9.73") into integer pennies
// (973), as required by the create/renew cost fields.
func priceToPennies(price string) (int64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(price), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid price %q", price)
	}
	return int64(math.Round(v * 100)), nil
}

// parsePorkbunTime parses the API's date formats ("2027-01-15 10:00:00" or
// "2027-01-15").
func parsePorkbunTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", value)
}

// envelopeError renders a Porkbun API error (envelope status ERROR or an
// HTTP-level failure) into an actionable error message.
func envelopeError(statusCode int, env *porkbunResponse, body []byte) error {
	if env.Status == "" && env.Message == "" {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 512 {
			snippet = snippet[:512] + "...(truncated)"
		}
		return fmt.Errorf("provider/porkbun: API returned HTTP %d: %s", statusCode, snippet)
	}

	msg := env.Message
	if msg == "" {
		msg = "unknown error"
	}
	if env.Code != "" {
		msg = fmt.Sprintf("%s (%s)", msg, env.Code)
	}
	if env.NextAction != nil && env.NextAction.Hint != "" {
		msg = fmt.Sprintf("%s; next: %s", msg, env.NextAction.Hint)
	}
	return fmt.Errorf("provider/porkbun: API error (HTTP %d): %s", statusCode, msg)
}

// do performs an authenticated request, retries once on 429 (honoring
// Retry-After), validates the response envelope, and decodes the body into
// out (when non-nil). It returns the HTTP status code of the final response.
func (c *PorkbunRegistrarClient) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return 0, fmt.Errorf("provider/porkbun: the registrar API requires api_key and api_secret (run 'indietool config add provider porkbun')")
	}

	for attempt := 0; ; attempt++ {
		status, retryAfter, env, data, err := c.attempt(ctx, method, path, body)
		if err != nil {
			return status, err
		}

		if status == http.StatusTooManyRequests && attempt == 0 {
			wait := retryAfter
			if wait <= 0 {
				wait = porkbunDefaultRetryWait
			}
			if wait > porkbunMaxRetryWait {
				wait = porkbunMaxRetryWait
			}
			select {
			case <-ctx.Done():
				return status, fmt.Errorf("provider/porkbun: request canceled while waiting out the rate limit: %w", ctx.Err())
			case <-time.After(wait):
			}
			continue
		}

		if status >= 400 || strings.EqualFold(env.Status, "ERROR") {
			return status, envelopeError(status, env, data)
		}

		if out != nil && len(data) > 0 {
			if err := json.Unmarshal(data, out); err != nil {
				return status, fmt.Errorf("provider/porkbun: failed to parse API response: %w", err)
			}
		}
		return status, nil
	}
}

// attempt issues a single HTTP request and returns the status code, the
// parsed Retry-After pause (429 only), the response envelope, and the raw
// body.
func (c *PorkbunRegistrarClient) attempt(ctx context.Context, method, path string, body any) (int, time.Duration, *porkbunResponse, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, 0, nil, nil, fmt.Errorf("provider/porkbun: failed to encode request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("provider/porkbun: failed to build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Secret-API-Key", c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("provider/porkbun: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, 0, nil, nil, fmt.Errorf("provider/porkbun: failed to read response: %w", err)
	}

	var env porkbunResponse
	_ = json.Unmarshal(data, &env) // envelope stays zero-valued on non-JSON bodies; handled by envelopeError

	var retryAfter time.Duration
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	}

	return resp.StatusCode, retryAfter, &env, data, nil
}

// parseRetryAfter parses a Retry-After header expressed in whole seconds.
func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// CheckDomain checks availability and pricing for a single domain
// (POST /domain/checkDomain/{domain} — the API takes one domain per call).
func (c *PorkbunRegistrarClient) CheckDomain(ctx context.Context, name string) (*porkbunCheckResponse, error) {
	var res porkbunCheckResponse
	if _, err := c.do(
		ctx,
		http.MethodPost,
		"/domain/checkDomain/"+url.PathEscape(name),
		nil,
		&res,
	); err != nil {
		return nil, err
	}
	return &res, nil
}

// CreateDomain registers a domain (POST /domain/create/{domain}). The call
// is synchronous: a SUCCESS response means the domain is registered and the
// account was charged `cost` pennies. With dryRun the API validates and
// returns a preview without creating an order or charging.
func (c *PorkbunRegistrarClient) CreateDomain(ctx context.Context, name string, costPennies int64, dryRun bool) (*porkbunCreateResponse, error) {
	var res porkbunCreateResponse
	if _, err := c.do(
		ctx,
		http.MethodPost,
		"/domain/create/"+url.PathEscape(name),
		&porkbunCreateRequest{
			Cost:         costPennies,
			AgreeToTerms: "yes",
			DryRun:       dryRun,
		},
		&res,
	); err != nil {
		return nil, err
	}
	return &res, nil
}

// UpdateAutoRenew toggles auto-renewal for a domain (POST
// /domain/updateAutoRenew/{domain}).
func (c *PorkbunRegistrarClient) UpdateAutoRenew(ctx context.Context, name string, enabled bool) error {
	status := "off"
	if enabled {
		status = "on"
	}

	var res porkbunResponse
	if _, err := c.do(
		ctx,
		http.MethodPost,
		"/domain/updateAutoRenew/"+url.PathEscape(name),
		&porkbunUpdateAutoRenewRequest{Status: status},
		&res,
	); err != nil {
		return err
	}
	return nil
}

// GetDomain fetches a single domain (GET /domain/get/{domain}). A 404 is
// returned as a "not found" error.
func (c *PorkbunRegistrarClient) GetDomain(ctx context.Context, name string) (*domains.ManagedDomain, error) {
	var res porkbunGetResponse
	status, err := c.do(
		ctx,
		http.MethodGet,
		"/domain/get/"+url.PathEscape(name),
		nil,
		&res,
	)
	if err != nil {
		if status == http.StatusNotFound {
			return nil, fmt.Errorf("provider/porkbun: domain %s not found", name)
		}
		return nil, err
	}
	return res.toManagedDomain(), nil
}
