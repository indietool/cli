package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// registrationFixtureJSON mirrors the new Registrar API registration resource
// (domain_name, status, created_at, expires_at, auto_renew, locked,
// privacy_mode). Fields absent from that schema are deliberately omitted.
func registrationFixtureJSON() string {
	return `{
		"domain_name": "example.dev",
		"status": "active",
		"created_at": "2024-01-01T00:00:00Z",
		"expires_at": "2027-03-01T00:00:00Z",
		"auto_renew": true,
		"locked": true,
		"privacy_mode": "redaction"
	}`
}

func writeDomainEnvelope(w http.ResponseWriter, result string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": ` + result + `}`))
}

func TestDomainsRenewShowsStatusWithoutPrice(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domains", "renew", "example.dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domains renew failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "example.dev") {
		t.Errorf("expected domain name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "2027-03-01") {
		t.Errorf("expected expiry in output, got:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "auto-renew") {
		t.Errorf("expected auto-renew status in output, got:\n%s", out)
	}
	// The new registrations schema carries no renewal price; it must not be
	// fabricated.
	if strings.Contains(out, "0.00") || strings.Contains(out, "USD") || strings.Contains(out, "10.11") {
		t.Errorf("expected no fabricated renewal price, got:\n%s", out)
	}
	if !strings.Contains(out, "not available via the Registrar API") {
		t.Errorf("expected the pricing-unavailable note, got:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "dashboard") {
		t.Errorf("expected the dashboard note, got:\n%s", out)
	}
}

func TestDomainsRenewToggleOn(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var gotMethod string
	var gotPath string
	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	mux.HandleFunc("/accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domains", "renew", "example.dev", "--on"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domains renew --on failed: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("expected a PATCH request to toggle auto-renew, got %q", gotMethod)
	}
	wantPath := "/accounts/" + testCmdAccountID + "/registrar/registrations/example.dev"
	if gotPath != wantPath {
		t.Errorf("expected PATCH path %q, got %q", wantPath, gotPath)
	}
	if !strings.Contains(gotBody, `"auto_renew":true`) {
		t.Errorf("expected auto_renew true in update body, got %q", gotBody)
	}
	if strings.Contains(gotBody, "locked") || strings.Contains(gotBody, "privacy") {
		t.Errorf("PATCH body must carry auto_renew only, got %q", gotBody)
	}
	if !strings.Contains(buf.String(), "example.dev") {
		t.Errorf("expected confirmation output, got:\n%s", buf.String())
	}
}

func TestDomainsRenewToggleOff(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	mux.HandleFunc("PATCH /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domains", "renew", "example.dev", "--off"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domains renew --off failed: %v", err)
	}

	if !strings.Contains(gotBody, `"auto_renew":false`) {
		t.Errorf("expected auto_renew false in update body, got %q", gotBody)
	}
}

func TestDomainsRenewJSONOmitsUnavailablePrice(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domains", "renew", "example.dev", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domains renew --json failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput:\n%s", err, buf.String())
	}
	if parsed["domain"] != "example.dev" {
		t.Errorf("unexpected JSON output: %+v", parsed)
	}
	if parsed["auto_renewal"] != true {
		t.Error("expected auto_renewal true in JSON output")
	}
	for _, key := range []string{"renewal_cost", "currency", "transfer_cost"} {
		if _, ok := parsed[key]; ok {
			t.Errorf("expected unavailable key %q to be omitted from JSON output", key)
		}
	}
}

func TestDomainsRenewRejectsConflictingFlags(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domains", "renew", "example.dev", "--on", "--off"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when both --on and --off are passed")
	}
}
