package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indietool/cli/domains"
)

const testAccountID = "test-account-id"

// newTestCloudflareProvider returns a CloudflareProvider whose registrar
// client is pointed at the given httptest server via the base URL override.
func newTestCloudflareProvider(t *testing.T, handler http.Handler) *CloudflareProvider {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv(CloudflareAPIBaseEnvVar, srv.URL)

	cfg := CloudflareConfig{
		AccountId: testAccountID,
		APIToken:  "test-token",
		Enabled:   true,
	}
	return NewCloudflare(cfg)
}

// cfEnvelope wraps a result in the standard Cloudflare API v4 envelope.
func cfEnvelope(t *testing.T, result any) []byte {
	return cfEnvelopeWithResultInfo(t, result, nil)
}

// cfEnvelopeWithResultInfo wraps a result plus optional result_info metadata
// (cursor pagination) in the standard Cloudflare API v4 envelope.
func cfEnvelopeWithResultInfo(t *testing.T, result any, resultInfo map[string]any) []byte {
	t.Helper()
	body := map[string]any{
		"success":  true,
		"errors":   []any{},
		"messages": []any{},
		"result":   result,
	}
	if resultInfo != nil {
		body["result_info"] = resultInfo
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}
	return data
}

// registrationFixture mirrors the new Registrar API registration resource:
// domain_name, status, created_at, expires_at, auto_renew, locked,
// privacy_mode. Fields absent from that schema (nameservers, renewal price)
// are deliberately omitted.
func registrationFixture() map[string]any {
	return map[string]any{
		"domain_name":  "example.dev",
		"status":       "active",
		"created_at":   "2024-01-01T00:00:00Z",
		"expires_at":   "2027-03-01T00:00:00Z",
		"auto_renew":   true,
		"locked":       true,
		"privacy_mode": "redaction",
	}
}

func TestGetDomain(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, registrationFixture()))
	})

	p := newTestCloudflareProvider(t, handler)

	dm, err := p.GetDomain(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("GetDomain returned error: %v", err)
	}

	if want := "/accounts/" + testAccountID + "/registrar/registrations/example.dev"; gotPath != want {
		t.Errorf("expected request path %q, got %q", want, gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET request, got %s", gotMethod)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Errorf("expected Bearer authorization, got %q", gotAuth)
	}

	if dm.Name != "example.dev" {
		t.Errorf("expected name example.dev, got %q", dm.Name)
	}
	if dm.Provider != "cloudflare" {
		t.Errorf("expected provider cloudflare, got %q", dm.Provider)
	}
	wantExpiry := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	if !dm.ExpiryDate.Equal(wantExpiry) {
		t.Errorf("expected expiry %v, got %v", wantExpiry, dm.ExpiryDate)
	}
	if !dm.AutoRenewal {
		t.Error("expected auto_renewal to be true")
	}
	if !dm.Locked {
		t.Error("expected locked to be true")
	}
	if !dm.Privacy {
		t.Error(`expected privacy to be true for privacy_mode "redaction"`)
	}

	// Fields absent from the new registrations schema must be dropped
	// gracefully, not fabricated.
	if len(dm.Nameservers) != 0 {
		t.Errorf("expected no nameservers from the registrations API, got %v", dm.Nameservers)
	}
	if dm.Cost != nil {
		t.Errorf("expected no renewal cost from the registrations API, got %+v", dm.Cost)
	}
}

func TestUpdateAutoRenewal(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, map[string]any{
			"state":     "succeeded",
			"completed": true,
		}))
	})

	p := newTestCloudflareProvider(t, handler)

	if err := p.UpdateAutoRenewal(context.Background(), "example.dev", true); err != nil {
		t.Fatalf("UpdateAutoRenewal(true) returned error: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("expected PATCH request, got %s", gotMethod)
	}
	if want := "/accounts/" + testAccountID + "/registrar/registrations/example.dev"; gotPath != want {
		t.Errorf("expected request path %q, got %q", want, gotPath)
	}
	if !strings.Contains(gotBody, `"auto_renew":true`) {
		t.Errorf("expected request body to set auto_renew true, got %q", gotBody)
	}
	if strings.Contains(gotBody, "locked") || strings.Contains(gotBody, "privacy") {
		t.Errorf("PATCH body must carry auto_renew only, got %q", gotBody)
	}

	if err := p.UpdateAutoRenewal(context.Background(), "example.dev", false); err != nil {
		t.Fatalf("UpdateAutoRenewal(false) returned error: %v", err)
	}
	if !strings.Contains(gotBody, `"auto_renew":false`) {
		t.Errorf("expected request body to set auto_renew false, got %q", gotBody)
	}
}

func TestUpdateAutoRenewalFailedWorkflow(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, map[string]any{
			"state":     "failed",
			"completed": true,
			"error": map[string]any{
				"code":    "internal_error",
				"message": "registry rejected the update",
			},
		}))
	})

	p := newTestCloudflareProvider(t, handler)

	err := p.UpdateAutoRenewal(context.Background(), "example.dev", false)
	if err == nil {
		t.Fatal("expected an error when the update workflow reports failed")
	}
	if !strings.Contains(err.Error(), "registry rejected the update") || !strings.Contains(err.Error(), "internal_error") {
		t.Errorf("expected the workflow error details, got: %v", err)
	}
}

func TestUpdateDomainSettingsRejectsUnsupported(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, registrationFixture()))
	})

	p := newTestCloudflareProvider(t, handler)

	// privacy only
	privacyOn := true
	err := p.UpdateDomainSettings(context.Background(), "example.dev", domains.DomainSettings{Privacy: &privacyOn})
	if err == nil {
		t.Fatal("expected an error when updating privacy")
	}
	if !strings.Contains(err.Error(), "not supported by the current Cloudflare Registrar API") {
		t.Errorf("expected unsupported-setting error, got: %v", err)
	}

	// locked only
	lockOff := false
	err = p.UpdateDomainSettings(context.Background(), "example.dev", domains.DomainSettings{Locked: &lockOff})
	if err == nil || !strings.Contains(err.Error(), "not supported by the current Cloudflare Registrar API") {
		t.Fatalf("expected unsupported-setting error for locked, got: %v", err)
	}

	// mixed auto-renew + privacy must be rejected without partial application
	autoRenewOff := false
	err = p.UpdateDomainSettings(context.Background(), "example.dev", domains.DomainSettings{AutoRenew: &autoRenewOff, Privacy: &autoRenewOff})
	if err == nil || !strings.Contains(err.Error(), "not supported by the current Cloudflare Registrar API") {
		t.Fatalf("expected unsupported-setting error for mixed settings, got: %v", err)
	}

	if n := atomic.LoadInt32(&requests); n != 0 {
		t.Errorf("expected zero HTTP calls for unsupported settings, got %d", n)
	}
}

func TestGetRenewalInfoUnavailable(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, registrationFixture()))
	})

	p := newTestCloudflareProvider(t, handler)

	cost, err := p.GetRenewalInfo(context.Background(), "example.dev")
	if cost != nil {
		t.Errorf("expected nil cost, got %+v", cost)
	}
	if !errors.Is(err, ErrRenewalPricingUnavailable) {
		t.Errorf("expected ErrRenewalPricingUnavailable, got: %v", err)
	}
	if n := atomic.LoadInt32(&requests); n != 0 {
		t.Errorf("expected zero HTTP calls when pricing is unavailable, got %d", n)
	}
}

func TestListDomainsPagination(t *testing.T) {
	var paths []string
	var cursors []string
	var perPages []string
	page := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		paths = append(paths, r.URL.Path)
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		perPages = append(perPages, r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			first := registrationFixture()
			second := registrationFixture()
			second["domain_name"] = "mybrand.dev"
			_, _ = w.Write(cfEnvelopeWithResultInfo(t, []any{first, second}, map[string]any{
				"count":    2,
				"cursor":   "next-page-cursor",
				"per_page": 50,
			}))
		default:
			third := registrationFixture()
			third["domain_name"] = "charlie.org"
			third["privacy_mode"] = "off"
			third["auto_renew"] = false
			_, _ = w.Write(cfEnvelopeWithResultInfo(t, []any{third}, map[string]any{
				"count":    1,
				"cursor":   "",
				"per_page": 50,
			}))
		}
	})

	p := newTestCloudflareProvider(t, handler)

	list, err := p.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("ListDomains returned error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paginated requests, got %d (%v)", len(paths), paths)
	}
	for i, path := range paths {
		if want := "/accounts/" + testAccountID + "/registrar/registrations"; path != want {
			t.Errorf("request %d: expected path %q, got %q", i+1, want, path)
		}
	}
	if cursors[0] != "" {
		t.Errorf("first request must not send a cursor, got %q", cursors[0])
	}
	if cursors[1] != "next-page-cursor" {
		t.Errorf("second request must follow the cursor, got %q", cursors[1])
	}
	for i, pp := range perPages {
		if pp != "50" {
			t.Errorf("request %d: expected per_page=50, got %q", i+1, pp)
		}
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 merged results across pages, got %d", len(list))
	}
	wantNames := []string{"example.dev", "mybrand.dev", "charlie.org"}
	for i, dm := range list {
		if dm.Name != wantNames[i] {
			t.Errorf("result %d: expected name %q, got %q", i, wantNames[i], dm.Name)
		}
		if dm.Provider != "cloudflare" {
			t.Errorf("result %d: expected provider cloudflare, got %q", i, dm.Provider)
		}
	}
	if list[2].Privacy {
		t.Error(`expected privacy false for privacy_mode "off"`)
	}
	if list[2].AutoRenewal {
		t.Error("expected auto_renewal false on page-2 result")
	}
}
