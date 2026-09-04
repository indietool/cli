package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/indietool/cli/providers"
)

// startFakePorkbunAPI starts a fake Porkbun API server and points the
// providers package at it via the base URL override.
func startFakePorkbunAPI(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv(providers.PorkbunAPIBaseEnvVar, srv.URL)
	return srv
}

// configurePorkbunForCmdTests adds an enabled Porkbun provider config to
// appConfig and persists it so initConfig builds a registry with it.
func configurePorkbunForCmdTests(t *testing.T, configPath string) {
	t.Helper()

	appConfig.Providers.Porkbun = &providers.PorkbunConfig{
		APIKey:    "pk1_test" + "key",
		APISecret: "sk1_test" + "secret",
		Enabled:   true,
	}
	if err := appConfig.SaveConfig(configPath); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}
}

// porkbunCheckFixture returns a SUCCESS checkDomain body for name/price.
func porkbunCheckFixture(avail, price, premium string) string {
	return `{"status": "SUCCESS", "response": {"avail": "` + avail + `", "price": "` + price +
		`", "premium": "` + premium + `", "additional": {"renewal": {"price": "11.00"}}}, "ttlRemaining": 0}`
}

func TestDomainCheckCommandPorkbun(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var gotAuthKey string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/example.com", func(w http.ResponseWriter, r *http.Request) {
		gotAuthKey = r.Header.Get("X-API-Key")
		_, _ = w.Write([]byte(porkbunCheckFixture("yes", "9.73", "no")))
	})
	mux.HandleFunc("POST /domain/checkDomain/taken.dev", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(porkbunCheckFixture("no", "", "no")))
	})
	startFakePorkbunAPI(t, mux)

	configPath := newTestConfigFile(t)
	configurePorkbunForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "check", "example.com", "taken.dev", "--provider", "porkbun", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain check --provider porkbun failed: %v", err)
	}

	if gotAuthKey == "" {
		t.Error("expected the X-API-Key auth header on the check request")
	}

	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput:\n%s", err, buf.String())
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 availability entries, got %d", len(parsed))
	}
	if parsed[0]["name"] != "example.com" || parsed[0]["registrable"] != true {
		t.Errorf("unexpected first entry: %v", parsed[0])
	}
	if parsed[0]["registration_cost"] != 9.73 {
		t.Errorf("expected registration_cost 9.73, got %v", parsed[0]["registration_cost"])
	}
	if parsed[1]["registrable"] != false {
		t.Errorf("expected taken.dev to be non-registrable: %v", parsed[1])
	}
}

func TestDomainCheckCommandAutoDetectsPorkbun(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var porkbunHits, cloudflareHits int32
	porkbunMux := http.NewServeMux()
	porkbunMux.HandleFunc("POST /domain/checkDomain/example.com", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&porkbunHits, 1)
		_, _ = w.Write([]byte(porkbunCheckFixture("yes", "9.73", "no")))
	})
	startFakePorkbunAPI(t, porkbunMux)

	cloudflareMux := http.NewServeMux()
	cloudflareMux.HandleFunc("POST /accounts/"+testCmdAccountID+"/registrar/domain-check", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cloudflareHits, 1)
		_, _ = w.Write([]byte(domainCheckFixture()))
	})
	srv := httptest.NewServer(cloudflareMux)
	t.Cleanup(srv.Close)
	t.Setenv(providers.CloudflareAPIBaseEnvVar, srv.URL)

	configPath := newTestConfigFile(t)
	configurePorkbunForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	// No --provider flag: the only configured purchase provider wins.
	rootCmd.SetArgs([]string{"domain", "check", "example.com", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain check auto-detect failed: %v", err)
	}

	if atomic.LoadInt32(&porkbunHits) != 1 {
		t.Errorf("expected the porkbun API to be queried, got %d hits", porkbunHits)
	}
	if atomic.LoadInt32(&cloudflareHits) != 0 {
		t.Errorf("expected no cloudflare calls, got %d hits", cloudflareHits)
	}
}

func TestDomainRegisterCommandPorkbun(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var checkCalls, createCalls int32
	var createCost struct {
		Cost         int64  `json:"cost"`
		AgreeToTerms string `json:"agreeToTerms"`
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/fresh.com", func(w http.ResponseWriter, r *http.Request) {
		// The CLI checks once for the price display; Register re-checks
		// internally to pin the exact cost.
		atomic.AddInt32(&checkCalls, 1)
		_, _ = w.Write([]byte(porkbunCheckFixture("yes", "9.73", "no")))
	})
	mux.HandleFunc("POST /domain/create/fresh.com", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&createCalls, 1)
		_ = json.NewDecoder(r.Body).Decode(&createCost)
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "domain": "fresh.com", "cost": 973, "orderId": 4242, "balance": 4054}`))
	})
	startFakePorkbunAPI(t, mux)

	configPath := newTestConfigFile(t)
	configurePorkbunForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "fresh.com", "--provider", "porkbun", "--yes"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("domain register --provider porkbun failed: %v", err)
	}

	if atomic.LoadInt32(&createCalls) != 1 {
		t.Errorf("expected exactly 1 create call, got %d", createCalls)
	}
	if atomic.LoadInt32(&checkCalls) != 2 {
		t.Errorf("expected 2 check calls (display + registration cost pin), got %d", checkCalls)
	}
	if createCost.Cost != 973 || createCost.AgreeToTerms != "yes" {
		t.Errorf("unexpected create payload: %+v", createCost)
	}

	out := buf.String()
	if !strings.Contains(out, "9.73") {
		t.Errorf("expected the price to be shown before registration, got:\n%s", out)
	}
	if !strings.Contains(out, "succeeded") {
		t.Errorf("expected the success message, got:\n%s", out)
	}
}

func TestDomainRegisterCommandPorkbunRefusesContact(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var createCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/create/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&createCalls, 1)
		_, _ = w.Write([]byte(`{"status": "SUCCESS"}`))
	})
	startFakePorkbunAPI(t, mux)

	configPath := newTestConfigFile(t)
	configurePorkbunForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "fresh.com", "--provider", "porkbun", "--yes",
		"--contact-name", "Jane Doe", "--contact-email", "jane@example.com",
		"--contact-phone", "+1.5555551234", "--contact-street", "1 Main St",
		"--contact-city", "Springfield", "--contact-state", "CA",
		"--contact-postal-code", "90210", "--contact-country", "US"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when contact flags are used with porkbun")
	}
	if !strings.Contains(err.Error(), "not supported for Porkbun") {
		t.Errorf("expected a porkbun contact refusal, got: %v", err)
	}
	if atomic.LoadInt32(&createCalls) != 0 {
		t.Errorf("no create call expected, got %d", createCalls)
	}
}

func TestDomainRegisterCommandPorkbunPremiumRefused(t *testing.T) {
	originalAppConfig := appConfig
	defer func() {
		appConfig = originalAppConfig
	}()

	var createCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/brand.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(porkbunCheckFixture("yes", "2999.00", "yes")))
	})
	mux.HandleFunc("POST /domain/create/brand.com", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&createCalls, 1)
		_, _ = w.Write([]byte(`{"status": "SUCCESS"}`))
	})
	startFakePorkbunAPI(t, mux)

	configPath := newTestConfigFile(t)
	configurePorkbunForCmdTests(t, configPath)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	rootCmd.SetArgs([]string{"domain", "register", "brand.com", "--provider", "porkbun", "--yes"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error for premium domains")
	}
	if !strings.Contains(err.Error(), "premium") {
		t.Errorf("expected a premium refusal, got: %v", err)
	}
	if atomic.LoadInt32(&createCalls) != 0 {
		t.Errorf("no create call expected for premium domains, got %d", createCalls)
	}
}
