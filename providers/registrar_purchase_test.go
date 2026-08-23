package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indietool/cli/domains"
)

func newTestPurchaseClient(t *testing.T, handler http.Handler) *RegistrarPurchaseClient {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := NewRegistrarPurchaseClient(testAccountID, "test-token", false)
	client.baseURL = srv.URL
	return client
}

func TestPurchaseCheck(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"domains": [
					{
						"name": "example.dev",
						"registrable": true,
						"tier": "standard",
						"pricing": {
							"currency": "USD",
							"registration_cost": "10.11",
							"renewal_cost": "10.11"
						}
					},
					{
						"name": "example.uk",
						"registrable": false,
						"reason": "extension_not_supported_via_api"
					}
				]
			}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	avail, err := client.Check(context.Background(), []string{"example.dev", "example.uk"})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}

	if want := "/accounts/" + testAccountID + "/registrar/domain-check"; gotPath != want {
		t.Errorf("expected path %q, got %q", want, gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Bearer auth header, got %q", gotAuth)
	}

	var reqBody struct {
		Domains []string `json:"domains"`
	}
	if err := json.Unmarshal(gotBody, &reqBody); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if len(reqBody.Domains) != 2 || reqBody.Domains[0] != "example.dev" {
		t.Errorf("unexpected request body domains: %v", reqBody.Domains)
	}

	if len(avail) != 2 {
		t.Fatalf("expected 2 availability results, got %d", len(avail))
	}

	dev := avail[0]
	if dev.Name != "example.dev" || !dev.Registrable {
		t.Errorf("unexpected first result: %+v", dev)
	}
	if dev.Currency != "USD" || dev.RegistrationCost != 10.11 || dev.RenewalCost != 10.11 {
		t.Errorf("unexpected pricing: %+v", dev)
	}

	uk := avail[1]
	if uk.Name != "example.uk" || uk.Registrable {
		t.Errorf("unexpected second result: %+v", uk)
	}
	if uk.Reason != "extension_not_supported_via_api" {
		t.Errorf("expected reason extension_not_supported_via_api, got %q", uk.Reason)
	}
}

func TestPurchaseCheckChunksRequests(t *testing.T) {
	var requestCount int
	var lastChunkSize int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Domains []string `json:"domains"`
		}
		_ = json.Unmarshal(body, &reqBody)
		lastChunkSize = len(reqBody.Domains)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {"domains": []}}`))
	})

	client := newTestPurchaseClient(t, handler)

	names := make([]string, 25)
	for i := range names {
		names[i] = "example" + string(rune('a'+i)) + ".dev"
	}

	if _, err := client.Check(context.Background(), names); err != nil {
		t.Fatalf("Check returned error: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 chunked requests for 25 domains, got %d", requestCount)
	}
	if lastChunkSize != 5 {
		t.Errorf("expected last chunk of 5 domains, got %d", lastChunkSize)
	}
}

func TestPurchaseRegisterCompleted(t *testing.T) {
	var gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"domain_name": "example.dev",
				"state": "succeeded",
				"completed": true
			}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	res, err := client.Register(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if want := "/accounts/" + testAccountID + "/registrar/registrations"; gotPath != want {
		t.Errorf("expected path %q, got %q", want, gotPath)
	}
	if res.DomainName != "example.dev" {
		t.Errorf("expected domain name example.dev, got %q", res.DomainName)
	}
	if res.State != domains.RegistrationStateSucceeded {
		t.Errorf("expected state succeeded, got %q", res.State)
	}
	if !res.Completed {
		t.Error("expected completed to be true")
	}
}

func TestPurchaseRegisterInProgress(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"domain_name": "example.dev",
				"state": "in_progress",
				"completed": false
			}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	res, err := client.Register(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if res.State != domains.RegistrationStateInProgress {
		t.Errorf("expected state in_progress, got %q", res.State)
	}
	if res.Completed {
		t.Error("expected completed to be false")
	}
	if res.IsTerminal() {
		t.Error("in_progress registration should not be terminal")
	}
}

func TestPurchaseRegisterSendsAsyncPreference(t *testing.T) {
	var gotPrefer string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPrefer = r.Header.Get("Prefer")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domain_name": "example.dev", "state": "in_progress", "completed": false}
		}`))
	})

	client := newTestPurchaseClient(t, handler)
	client.PreferAsync = true

	if _, err := client.Register(context.Background(), "example.dev"); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if gotPrefer != "respond-async" {
		t.Errorf("expected Prefer: respond-async header, got %q", gotPrefer)
	}
}

func TestPurchaseRegistrationStatus(t *testing.T) {
	var gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"domain_name": "example.dev",
				"state": "action_required",
				"completed": false,
				"error": {
					"code": "contact_verification_required",
					"message": "Verify the registrant contact in the Cloudflare dashboard"
				}
			}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	res, err := client.RegistrationStatus(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("RegistrationStatus returned error: %v", err)
	}

	if want := "/accounts/" + testAccountID + "/registrar/registrations/example.dev/registration-status"; gotPath != want {
		t.Errorf("expected path %q, got %q", want, gotPath)
	}
	if res.State != domains.RegistrationStateActionRequired {
		t.Errorf("expected state action_required, got %q", res.State)
	}
	if res.Error == nil || !strings.Contains(*res.Error, "contact_verification_required") {
		t.Errorf("expected error to surface the action required code, got %+v", res.Error)
	}
}

func TestPurchaseAPIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"success": false,
			"errors": [
				{"code": 10000, "message": "domain_name is required"}
			],
			"messages": [],
			"result": null
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	_, err := client.Register(context.Background(), "example.dev")
	if err == nil {
		t.Fatal("expected an error for an unsuccessful API response")
	}
	if !strings.Contains(err.Error(), "domain_name is required") {
		t.Errorf("expected error to contain API message, got: %v", err)
	}
}

func TestPurchaseGetRegistration(t *testing.T) {
	var gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {
				"domain_name": "example.dev",
				"status": "active",
				"created_at": "2024-01-01T00:00:00Z",
				"expires_at": null,
				"auto_renew": false,
				"locked": false,
				"privacy_mode": "off"
			}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	dm, err := client.GetRegistration(context.Background(), "example.dev")
	if err != nil {
		t.Fatalf("GetRegistration returned error: %v", err)
	}

	if want := "/accounts/" + testAccountID + "/registrar/registrations/example.dev"; gotPath != want {
		t.Errorf("expected path %q, got %q", want, gotPath)
	}
	if dm.Name != "example.dev" {
		t.Errorf("expected domain_name to map to Name, got %q", dm.Name)
	}
	if !dm.ExpiryDate.IsZero() {
		t.Errorf("expected null expires_at to leave expiry at the zero value, got %v", dm.ExpiryDate)
	}
	if dm.AutoRenewal || dm.Locked || dm.Privacy {
		t.Errorf("expected off/false management fields, got %+v", dm)
	}
	if dm.Cost != nil || len(dm.Nameservers) != 0 {
		t.Errorf("fields absent from the new schema must not be fabricated: %+v", dm)
	}
}

func TestPurchaseGetRegistrationEscapesName(t *testing.T) {
	var gotEscapedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEscapedPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {}}`))
	})

	client := newTestPurchaseClient(t, handler)

	if _, err := client.GetRegistration(context.Background(), "../other"); err != nil {
		t.Fatalf("GetRegistration returned error: %v", err)
	}
	// The "/" must be percent-encoded so the input stays a single path segment.
	want := "/accounts/" + testAccountID + "/registrar/registrations/..%2Fother"
	if gotEscapedPath != want {
		t.Errorf("expected the domain name to be path-escaped to %q, got %q", want, gotEscapedPath)
	}
}

func TestPurchaseUpdateAutoRenewSendsPatchBody(t *testing.T) {
	var gotMethod, gotBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"state": "in_progress", "completed": false}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	if err := client.UpdateAutoRenew(context.Background(), "example.dev", true); err != nil {
		t.Fatalf("UpdateAutoRenew returned error: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("expected PATCH request, got %s", gotMethod)
	}
	want := `{"auto_renew":true}`
	if strings.TrimSpace(gotBody) != want {
		t.Errorf("expected body %s, got %s", want, gotBody)
	}
}

func TestPurchaseUpdateAutoRenewAcceptsPendingWorkflow(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"state": "pending", "completed": false}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	if err := client.UpdateAutoRenew(context.Background(), "example.dev", false); err != nil {
		t.Fatalf("expected pending workflow status to be accepted, got error: %v", err)
	}
}

func TestPurchaseListRegistrationsFollowsCursor(t *testing.T) {
	var cursors []string
	page := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		page++
		w.Header().Set("Content-Type", "application/json")
		if page == 1 {
			_, _ = w.Write([]byte(`{
				"success": true, "errors": [], "messages": [],
				"result": [{"domain_name": "a.dev"}],
				"result_info": {"count": 1, "cursor": "c2", "per_page": 50}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": [{"domain_name": "b.dev"}, {"domain_name": "c.dev"}],
			"result_info": {"count": 2, "cursor": "", "per_page": 50}
		}`))
	})

	client := newTestPurchaseClient(t, handler)

	list, err := client.ListRegistrations(context.Background())
	if err != nil {
		t.Fatalf("ListRegistrations returned error: %v", err)
	}

	if len(cursors) != 2 || cursors[0] != "" || cursors[1] != "c2" {
		t.Errorf("unexpected cursor sequence: %v", cursors)
	}
	if len(list) != 3 || list[0].Name != "a.dev" || list[2].Name != "c.dev" {
		t.Errorf("unexpected merged list: %+v", list)
	}
}
