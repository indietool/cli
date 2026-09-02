package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indietool/cli/providers"
)

const testCmdAccountID = "acc-123"

// startFakeCloudflareAPI starts a fake Cloudflare API server and points the
// providers package at it via the base URL override.
func startFakeCloudflareAPI(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv(providers.CloudflareAPIBaseEnvVar, srv.URL)
	return srv
}

// configureCloudflareForCmdTests adds an enabled Cloudflare provider config to
// appConfig and persists it so initConfig builds a registry with it.
func configureCloudflareForCmdTests(t *testing.T, configPath string) {
	t.Helper()

	appConfig.Providers.Cloudflare = &providers.CloudflareConfig{
		AccountId: testCmdAccountID,
		APIToken:  "test-token",
		Enabled:   true,
	}
	if err := appConfig.SaveConfig(configPath); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}
}

func domainCheckFixture() string {
	return `{
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
	}`
}

func TestDomainCheckCommand(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(domainCheckFixture()))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "check", "example.dev", "example.uk"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain check failed: %v", err)
	}

	if gotPath != "/accounts/"+testCmdAccountID+"/registrar/domain-check" {
		t.Errorf("unexpected request path: %q", gotPath)
	}

	out := buf.String()
	if !strings.Contains(out, "example.dev") {
		t.Errorf("expected output to contain example.dev, got:\n%s", out)
	}
	if !strings.Contains(out, "example.uk") {
		t.Errorf("expected output to contain example.uk, got:\n%s", out)
	}
	if !strings.Contains(out, "extension_not_supported_via_api") {
		t.Errorf("expected output to surface the non-registrable reason, got:\n%s", out)
	}
	if !strings.Contains(out, "10.11") {
		t.Errorf("expected output to contain the registration price, got:\n%s", out)
	}
}

func TestDomainCheckCommandJSON(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(domainCheckFixture()))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "check", "example.dev", "example.uk", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain check --json failed: %v", err)
	}

	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput:\n%s", err, buf.String())
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 availability entries, got %d", len(parsed))
	}
	if parsed[0]["name"] != "example.dev" {
		t.Errorf("unexpected first entry: %v", parsed[0])
	}
	if parsed[0]["registration_cost"] != 10.11 {
		t.Errorf("expected registration_cost 10.11, got %v", parsed[0]["registration_cost"])
	}
}

func TestDomainCheckCommandRequiresCloudflare(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	newTestConfigFile(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "check", "example.dev"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when cloudflare is not configured")
	}
	if !strings.Contains(err.Error(), "cloudflare") {
		t.Errorf("expected error to mention cloudflare, got: %v", err)
	}
}
