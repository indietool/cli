package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDomainRegisterCommandWithYes(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	originalInterval := registerPollInterval
	registerPollInterval = time.Millisecond
	defer func() { registerPollInterval = originalInterval }()

	var registerCalls, statusCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domains": [{
				"name": "example.dev",
				"registrable": true,
				"tier": "standard",
				"pricing": {"currency": "USD", "registration_cost": "10.11", "renewal_cost": "10.11"}
			}]}
		}`))
	})
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/registrations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&registerCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domain_name": "example.dev", "state": "in_progress", "completed": false}
		}`))
	})
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev/registration-status", func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&statusCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		if call == 1 {
			_, _ = w.Write([]byte(`{
				"success": true, "errors": [], "messages": [],
				"result": {"domain_name": "example.dev", "state": "in_progress", "completed": false}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domain_name": "example.dev", "state": "succeeded", "completed": true}
		}`))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "example.dev", "--yes"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain register failed: %v", err)
	}

	if atomic.LoadInt32(&registerCalls) != 1 {
		t.Errorf("expected exactly 1 registration call, got %d", registerCalls)
	}
	if atomic.LoadInt32(&statusCalls) < 2 {
		t.Errorf("expected the status endpoint to be polled until terminal state, got %d calls", statusCalls)
	}

	out := buf.String()
	if !strings.Contains(out, "10.11") {
		t.Errorf("expected the price to be shown before registration, got:\n%s", out)
	}
	if !strings.Contains(out, "succeeded") {
		t.Errorf("expected the terminal state in the output, got:\n%s", out)
	}
}

func TestDomainRegisterCommandStopsOnActionRequired(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	originalInterval := registerPollInterval
	registerPollInterval = time.Millisecond
	defer func() { registerPollInterval = originalInterval }()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domains": [{
				"name": "example.dev",
				"registrable": true,
				"pricing": {"currency": "USD", "registration_cost": "10.11", "renewal_cost": "10.11"}
			}]}
		}`))
	})
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/registrations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domain_name": "example.dev", "state": "in_progress", "completed": false}
		}`))
	})
	mux.HandleFunc("GET /accounts/"+testCmdAccountID+"/registrar/registrations/example.dev/registration-status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {
				"domain_name": "example.dev",
				"state": "action_required",
				"completed": false,
				"error": {"code": "contact_verification_required", "message": "Verify the registrant contact in the Cloudflare dashboard"}
			}
		}`))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "example.dev", "--yes"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when registration requires action")
	}
	if !strings.Contains(err.Error(), "Verify the registrant contact") {
		t.Errorf("expected actionable message in error, got: %v", err)
	}
}

func TestDomainRegisterCommandRequiresConfirmation(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var registerCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domains": [{
				"name": "example.dev",
				"registrable": true,
				"pricing": {"currency": "USD", "registration_cost": "10.11", "renewal_cost": "10.11"}
			}]}
		}`))
	})
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/registrations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&registerCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domain_name": "example.dev", "state": "succeeded", "completed": true}
		}`))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader("n\n"))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)
	defer rootCmd.SetIn(nil)

	rootCmd.SetArgs([]string{"domain", "register", "example.dev"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected registration to be aborted without confirmation")
	}
	if atomic.LoadInt32(&registerCalls) != 0 {
		t.Errorf("expected no registration call without confirmation, got %d", registerCalls)
	}
	if !strings.Contains(buf.String(), "10.11") {
		t.Errorf("expected the price to be shown before the prompt, got:\n%s", buf.String())
	}
}

func TestDomainRegisterCommandDryRun(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var registerCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domains": [{
				"name": "example.dev",
				"registrable": true,
				"pricing": {"currency": "USD", "registration_cost": "10.11", "renewal_cost": "10.11"}
			}]}
		}`))
	})
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/registrations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&registerCalls, 1)
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "example.dev", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain register --dry-run failed: %v", err)
	}
	if atomic.LoadInt32(&registerCalls) != 0 {
		t.Errorf("expected no registration call for --dry-run, got %d", registerCalls)
	}
	if !strings.Contains(buf.String(), "10.11") {
		t.Errorf("expected the price in dry-run output, got:\n%s", buf.String())
	}
}

func TestDomainRegisterCommandRejectsNonRegistrable(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domains": [{
				"name": "example.uk",
				"registrable": false,
				"reason": "extension_not_supported_via_api"
			}]}
		}`))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "example.uk", "--yes"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error for a non-registrable domain")
	}
	if !strings.Contains(err.Error(), "extension_not_supported_via_api") {
		t.Errorf("expected the reason in the error, got: %v", err)
	}
}

func TestDomainRegisterCommandPartialContactFlags(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "example.dev", "--yes", "--contact-name", "Jane Doe"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error for incomplete contact flags")
	}
	if !strings.Contains(err.Error(), "--contact-email") {
		t.Errorf("expected the missing flags to be named, got: %v", err)
	}
}

func TestDomainRegisterCommandWithContactFlags(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	originalInterval := registerPollInterval
	registerPollInterval = time.Millisecond
	defer func() { registerPollInterval = originalInterval }()

	var registerCalls int32
	contactCh := make(chan map[string]any, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true, "errors": [], "messages": [],
			"result": {"domains": [{
				"name": "example.dev",
				"registrable": true,
				"pricing": {"currency": "USD", "registration_cost": "10.11", "renewal_cost": "10.11"}
			}]}
		}`))
	})
	mux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/registrations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&registerCalls, 1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode registration body: %v", err)
		}
		contactCh <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true, "errors": [], "messages": [], "result": {"domain_name": "example.dev", "state": "succeeded", "completed": true}}`))
	})
	startFakeCloudflareAPI(t, mux)

	configPath := newTestConfigFile(t)
	configureCloudflareForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{
		"domain", "register", "example.dev", "--yes",
		"--contact-name", "Jane Doe",
		"--contact-email", "jane@example.com",
		"--contact-phone", "+1.5555551234",
		"--contact-street", "1 Main St",
		"--contact-city", "Springfield",
		"--contact-state", "CA",
		"--contact-postal-code", "90210",
		"--contact-country", "us",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain register with contact flags failed: %v", err)
	}
	if atomic.LoadInt32(&registerCalls) != 1 {
		t.Errorf("expected exactly 1 registration call, got %d", registerCalls)
	}

	var gotContact map[string]any
	select {
	case gotContact = <-contactCh:
	case <-time.After(time.Second):
		t.Fatal("no registration body captured")
	}

	contacts, _ := gotContact["contacts"].(map[string]any)
	reg, _ := contacts["registrant"].(map[string]any)
	postal, _ := reg["postal_info"].(map[string]any)
	addr, _ := postal["address"].(map[string]any)
	if addr == nil || addr["country_code"] != "US" {
		t.Errorf("expected normalized country_code US in the registration payload, got: %v", gotContact)
	}
	if reg == nil || reg["phone"] != "+1.5555551234" || reg["email"] != "jane@example.com" {
		t.Errorf("unexpected registrant contact in payload: %v", reg)
	}
	if postal == nil || postal["name"] != "Jane Doe" {
		t.Errorf("unexpected postal name in payload: %v", postal)
	}
}
