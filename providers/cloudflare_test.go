package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/option"
)

const testAccountID = "test-account-id"

// newTestCloudflareProvider returns a CloudflareProvider whose SDK client is
// pointed at the given httptest server.
func newTestCloudflareProvider(t *testing.T, handler http.Handler) *CloudflareProvider {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := CloudflareConfig{
		AccountId: testAccountID,
		APIToken:  "test-token",
		Enabled:   true,
	}

	p := NewCloudflare(cfg)
	p.client = cloudflare.NewClient(
		option.WithAPIToken("test-token"),
		option.WithBaseURL(srv.URL),
	)
	return p
}

// cfEnvelope wraps a result in the standard Cloudflare API v4 envelope.
func cfEnvelope(t *testing.T, result any) []byte {
	t.Helper()
	body := map[string]any{
		"success":  true,
		"errors":   []any{},
		"messages": []any{},
		"result":   result,
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}
	return data
}

func domainGetFixture() map[string]any {
	return map[string]any{
		"id":                 "example.dev",
		"name":               "example.dev",
		"available":          false,
		"can_register":       false,
		"created_at":         "2024-01-01T00:00:00Z",
		"current_registrar":  "Cloudflare, Inc.",
		"expires_at":         "2027-03-01T00:00:00Z",
		"locked":             true,
		"privacy":            true,
		"auto_renew":         true,
		"renewal_price":      10.11,
		"name_servers":       []string{"ara.ns.cloudflare.com", "duke.ns.cloudflare.com"},
		"registry_statuses":  "ok,active",
		"supported_tld":      true,
		"updated_at":         "2025-01-01T00:00:00Z",
	}
}

func TestGetDomain(t *testing.T) {
	var gotPath, gotAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, domainGetFixture()))
	})

	p := newTestCloudflareProvider(t, handler)

	dm, err := p.GetDomain(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("GetDomain returned error: %v", err)
	}

	if want := "/accounts/" + testAccountID + "/registrar/domains/example.dev"; gotPath != want {
		t.Errorf("expected request path %q, got %q", want, gotPath)
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
		t.Error("expected privacy to be true")
	}
	if len(dm.Nameservers) != 2 || dm.Nameservers[0] != "ara.ns.cloudflare.com" {
		t.Errorf("unexpected nameservers: %v", dm.Nameservers)
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
		_, _ = w.Write(cfEnvelope(t, domainGetFixture()))
	})

	p := newTestCloudflareProvider(t, handler)

	if err := p.UpdateAutoRenewal(context.Background(), "example.dev", true); err != nil {
		t.Fatalf("UpdateAutoRenewal returned error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT request, got %s", gotMethod)
	}
	if want := "/accounts/" + testAccountID + "/registrar/domains/example.dev"; gotPath != want {
		t.Errorf("expected request path %q, got %q", want, gotPath)
	}
	if !strings.Contains(gotBody, `"auto_renew":true`) {
		t.Errorf("expected request body to set auto_renew true, got %q", gotBody)
	}

	if err := p.UpdateAutoRenewal(context.Background(), "example.dev", false); err != nil {
		t.Fatalf("UpdateAutoRenewal returned error: %v", err)
	}
	if !strings.Contains(gotBody, `"auto_renew":false`) {
		t.Errorf("expected request body to set auto_renew false, got %q", gotBody)
	}
}

func TestGetRenewalInfo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfEnvelope(t, domainGetFixture()))
	})

	p := newTestCloudflareProvider(t, handler)

	cost, err := p.GetRenewalInfo(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("GetRenewalInfo returned error: %v", err)
	}

	if cost.Currency != "USD" {
		t.Errorf("expected currency USD, got %q", cost.Currency)
	}
	if cost.RenewalPrice != 10.11 {
		t.Errorf("expected renewal price 10.11, got %v", cost.RenewalPrice)
	}
	if cost.TransferPrice != 0 {
		t.Errorf("expected transfer price 0, got %v", cost.TransferPrice)
	}
}
