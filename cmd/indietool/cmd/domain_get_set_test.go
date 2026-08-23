package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDomainGetShowsDetails(t *testing.T) {
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

	rootCmd.SetArgs([]string{"domain", "get", "example.dev"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain get failed: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"example.dev", "2027-03-01"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
	lower := strings.ToLower(out)
	for _, want := range []string{"auto-renew", "locked", "privacy"} {
		if !strings.Contains(lower, want) {
			t.Errorf("expected output to mention %q, got:\n%s", want, out)
		}
	}
	// Fields absent from the new registrations schema must not be fabricated.
	if strings.Contains(out, "ara.ns.cloudflare.com") {
		t.Error("expected no nameservers in output (absent from the registrations schema)")
	}
	if strings.Contains(out, "10.11") || strings.Contains(out, "USD") {
		t.Error("expected no renewal price in output (absent from the registrations schema)")
	}
}

func TestDomainGetJSON(t *testing.T) {
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

	rootCmd.SetArgs([]string{"domain", "get", "example.dev", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain get --json failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput:\n%s", err, buf.String())
	}
	if parsed["name"] != "example.dev" {
		t.Errorf("unexpected JSON output: %v", parsed)
	}
	if parsed["auto_renewal"] != true {
		t.Errorf("expected auto_renewal true, got %v", parsed["auto_renewal"])
	}
	if parsed["locked"] != true {
		t.Errorf("expected locked true, got %v", parsed["locked"])
	}
	if parsed["privacy"] != true {
		t.Errorf("expected privacy true, got %v", parsed["privacy"])
	}
}

func TestDomainSetAutoRenewOn(t *testing.T) {
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

	rootCmd.SetArgs([]string{"domain", "set", "example.dev", "--auto-renew", "--on"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain set failed: %v", err)
	}

	if !strings.Contains(gotBody, `"auto_renew":true`) {
		t.Errorf("expected auto_renew true in update body, got %q", gotBody)
	}
	if strings.Contains(gotBody, "privacy") || strings.Contains(gotBody, "locked") {
		t.Errorf("expected only auto_renew in update body, got %q", gotBody)
	}
}

func TestDomainSetPrivacyAndLockedFailFast(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	// Privacy/lock have no equivalent in the new Registrar API (PATCH
	// supports auto_renew only): the command must fail fast with a clear
	// message and must not send any mutation request.
	var mutationCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		writeDomainEnvelope(w, registrationFixtureJSON())
	})
	mux.HandleFunc("/accounts/"+testCmdAccountID+"/registrar/registrations/example.dev", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			atomic.AddInt32(&mutationCalls, 1)
		}
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

	rootCmd.SetArgs([]string{"domain", "set", "example.dev", "--privacy", "--locked", "--off"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when updating privacy/lock via the API")
	}
	if !strings.Contains(err.Error(), "not supported by the current Cloudflare Registrar API") {
		t.Errorf("expected unsupported-setting error, got: %v", err)
	}
	if n := atomic.LoadInt32(&mutationCalls); n != 0 {
		t.Errorf("expected no mutation requests for privacy/lock, got %d", n)
	}
}

func TestDomainSetRequiresSettingFlag(t *testing.T) {
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

	rootCmd.SetArgs([]string{"domain", "set", "example.dev", "--on"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when no setting flag is passed")
	}
}
