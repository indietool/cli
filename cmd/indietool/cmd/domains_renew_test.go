package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func domainFixtureJSON() string {
	return `{
		"id": "example.dev",
		"name": "example.dev",
		"available": false,
		"can_register": false,
		"created_at": "2024-01-01T00:00:00Z",
		"current_registrar": "Cloudflare, Inc.",
		"expires_at": "2027-03-01T00:00:00Z",
		"locked": true,
		"privacy": true,
		"auto_renew": true,
		"renewal_price": 10.11,
		"name_servers": ["ara.ns.cloudflare.com", "duke.ns.cloudflare.com"],
		"registry_statuses": "ok,active",
		"supported_tld": true,
		"updated_at": "2025-01-01T00:00:00Z"
	}`
}

func writeDomainEnvelope(w http.ResponseWriter, result string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": ` + result + `}`))
}

func TestDomainsRenewShowsPriceAndStatus(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, domainFixtureJSON())
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
	if !strings.Contains(out, "10.11") {
		t.Errorf("expected renewal price in output, got:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "auto-renew") {
		t.Errorf("expected auto-renew status in output, got:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "dashboard") {
		t.Errorf("expected the manual-renewal dashboard note, got:\n%s", out)
	}
}

func TestDomainsRenewToggleOn(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var gotMethod string
	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, domainFixtureJSON())
	})
	mux.HandleFunc("PUT /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		writeDomainEnvelope(w, domainFixtureJSON())
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

	if gotMethod != http.MethodPut {
		t.Errorf("expected a PUT request to toggle auto-renew, got %q", gotMethod)
	}
	if !strings.Contains(gotBody, `"auto_renew":true`) {
		t.Errorf("expected auto_renew true in update body, got %q", gotBody)
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
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, domainFixtureJSON())
	})
	mux.HandleFunc("PUT /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		writeDomainEnvelope(w, domainFixtureJSON())
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

func TestDomainsRenewJSON(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, domainFixtureJSON())
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

	var parsed struct {
		Domain      string  `json:"domain"`
		AutoRenewal bool    `json:"auto_renewal"`
		RenewalCost float64 `json:"renewal_cost"`
		Currency    string  `json:"currency"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput:\n%s", err, buf.String())
	}
	if parsed.Domain != "example.dev" {
		t.Errorf("unexpected JSON output: %+v", parsed)
	}
	if parsed.RenewalCost != 10.11 {
		t.Errorf("expected renewal_cost 10.11, got %v", parsed.RenewalCost)
	}
	if !parsed.AutoRenewal {
		t.Error("expected auto_renewal true in JSON output")
	}
}

func TestDomainsRenewRejectsConflictingFlags(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/domains/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, domainFixtureJSON())
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
