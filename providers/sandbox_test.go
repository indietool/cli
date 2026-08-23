package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testPurchaseToken = "test-" + "token" // assembled to avoid secret-scan false positives

// newTestSandboxPurchaseClient builds a purchase client with sandbox mode
// enabled, asserting that all registrar requests go to /registrar-sandbox/.
func newTestSandboxPurchaseClient(t *testing.T, handler http.Handler) *RegistrarPurchaseClient {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := NewRegistrarPurchaseClient(testAccountID, testPurchaseToken, true)
	client.baseURL = srv.URL
	return client
}

// TestPurchaseSandboxPathPrefix verifies that every registrar purchase/management
// call routes through the /registrar-sandbox/ path prefix when sandbox mode is
// enabled, mirroring Cloudflare's Registrar Sandbox API exactly.
func TestPurchaseSandboxPathPrefix(t *testing.T) {
	var paths []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/domain-check"):
			_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {"domains": []}}`))
		case strings.HasSuffix(r.URL.Path, "/registration-status"):
			_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {"domain_name": "example.dev", "state": "succeeded", "completed": true}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/registrations"):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {"domain_name": "example.dev", "state": "succeeded", "completed": true}}`))
		case r.Method == http.MethodPatch:
			_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {"state": "in_progress", "completed": false}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/registrations"):
			_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": [], "result_info": {"count": 0, "cursor": "", "per_page": 50}}`))
		default:
			_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {}}`))
		}
	})

	client := newTestSandboxPurchaseClient(t, handler)

	if _, err := client.Check(context.Background(), []string{"example.dev"}); err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if _, err := client.Register(context.Background(), "example.dev"); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if _, err := client.RegistrationStatus(context.Background(), "example.dev"); err != nil {
		t.Fatalf("RegistrationStatus returned error: %v", err)
	}
	if _, err := client.GetRegistration(context.Background(), "example.dev"); err != nil {
		t.Fatalf("GetRegistration returned error: %v", err)
	}
	if err := client.UpdateAutoRenew(context.Background(), "example.dev", true); err != nil {
		t.Fatalf("UpdateAutoRenew returned error: %v", err)
	}
	if _, err := client.ListRegistrations(context.Background()); err != nil {
		t.Fatalf("ListRegistrations returned error: %v", err)
	}

	want := []string{
		"/accounts/" + testAccountID + "/registrar-sandbox/domain-check",
		"/accounts/" + testAccountID + "/registrar-sandbox/registrations",
		"/accounts/" + testAccountID + "/registrar-sandbox/registrations/example.dev/registration-status",
		"/accounts/" + testAccountID + "/registrar-sandbox/registrations/example.dev",
		"/accounts/" + testAccountID + "/registrar-sandbox/registrations/example.dev",
		"/accounts/" + testAccountID + "/registrar-sandbox/registrations",
	}
	if len(paths) != len(want) {
		t.Fatalf("expected %d requests, got %d: %v", len(want), len(paths), paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("request %d: expected path %q, got %q", i, want[i], paths[i])
		}
	}
}
