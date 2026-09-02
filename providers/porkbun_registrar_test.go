package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indietool/cli/domains"
)

const (
	testPorkbunAPIKey    = "pk1_test" + "key"
	testPorkbunAPISecret = "sk1_test" + "secret"
)

// newTestPorkbunClient builds a registrar client pointed at a fake API
// server via the base URL override.
func newTestPorkbunClient(t *testing.T, handler http.Handler) *PorkbunRegistrarClient {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv(PorkbunAPIBaseEnvVar, srv.URL)

	return NewPorkbunRegistrarClient(testPorkbunAPIKey, testPorkbunAPISecret)
}

// newTestPorkbunProvider builds a PorkbunProvider with credentials, pointed
// at the same fake API server as newTestPorkbunClient.
func newTestPorkbunProvider(t *testing.T, handler http.Handler) *PorkbunProvider {
	newTestPorkbunClient(t, handler)
	return NewPorkbun(PorkbunConfig{
		APIKey:    testPorkbunAPIKey,
		APISecret: testPorkbunAPISecret,
		Enabled:   true,
	})
}

func TestPorkbunCheckDomainAvailable(t *testing.T) {
	var gotPath string
	var gotKey, gotSecret string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/example.com", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-API-Key")
		gotSecret = r.Header.Get("X-Secret-API-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "SUCCESS",
			"response": {
				"avail": "yes",
				"type": "registration",
				"price": "9.73",
				"firstYearPromo": "yes",
				"regularPrice": "11.00",
				"premium": "no",
				"minDuration": 1,
				"additional": {
					"renewal": {"type": "renewal", "price": "12.34"},
					"transfer": {"type": "transfer", "price": "9.73"}
				}
			},
			"ttlRemaining": 0
		}`))
	})

	client := newTestPorkbunClient(t, mux)
	res, err := client.CheckDomain(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("CheckDomain failed: %v", err)
	}

	if gotPath != "/domain/checkDomain/example.com" {
		t.Errorf("unexpected request path: %q", gotPath)
	}
	if gotKey != testPorkbunAPIKey || gotSecret != testPorkbunAPISecret {
		t.Errorf("auth headers missing or wrong: key=%q secret=%q", gotKey, gotSecret)
	}

	avail := res.toAvailability("example.com")
	if !avail.Registrable {
		t.Errorf("expected registrable, got %+v", avail)
	}
	if avail.RegistrationCost != 9.73 {
		t.Errorf("expected registration cost 9.73, got %v", avail.RegistrationCost)
	}
	if avail.RenewalCost != 12.34 {
		t.Errorf("expected renewal cost 12.34, got %v", avail.RenewalCost)
	}
	if avail.Currency != "USD" {
		t.Errorf("expected USD, got %q", avail.Currency)
	}
	if avail.Tier != "first-year-promo" {
		t.Errorf("expected first-year-promo tier, got %q", avail.Tier)
	}
}

func TestPorkbunCheckDomainUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/taken.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "no"}, "ttlRemaining": 0}`))
	})

	client := newTestPorkbunClient(t, mux)
	res, err := client.CheckDomain(context.Background(), "taken.com")
	if err != nil {
		t.Fatalf("CheckDomain failed: %v", err)
	}

	avail := res.toAvailability("taken.com")
	if avail.Registrable {
		t.Errorf("expected not registrable, got %+v", avail)
	}
	if avail.Reason != "unavailable" {
		t.Errorf("expected unavailable reason, got %q", avail.Reason)
	}
}

func TestPorkbunCheckDomainPremiumNotRegistrable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/premium.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "yes", "premium": "yes", "price": "2999.00"}, "ttlRemaining": 0}`))
	})

	client := newTestPorkbunClient(t, mux)
	res, err := client.CheckDomain(context.Background(), "premium.com")
	if err != nil {
		t.Fatalf("CheckDomain failed: %v", err)
	}

	avail := res.toAvailability("premium.com")
	if avail.Registrable {
		t.Errorf("premium domains must not be reported registrable, got %+v", avail)
	}
	if avail.Tier != "premium" {
		t.Errorf("expected premium tier, got %q", avail.Tier)
	}
	if !strings.Contains(avail.Reason, "premium") {
		t.Errorf("expected premium reason, got %q", avail.Reason)
	}
}

func TestPorkbunErrorEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/bad.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"status": "ERROR",
			"message": "Invalid domain.",
			"code": "INVALID_DOMAIN",
			"next_action": {"type": "fix_request", "hint": "Send a valid domain name."}
		}`))
	})

	client := newTestPorkbunClient(t, mux)
	_, err := client.CheckDomain(context.Background(), "bad.com")
	if err == nil {
		t.Fatal("expected an error for the ERROR envelope")
	}
	for _, want := range []string{"HTTP 400", "Invalid domain.", "INVALID_DOMAIN", "Send a valid domain name."} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got: %v", want, err)
		}
	}
}

func TestPorkbunRateLimitRetryAfter(t *testing.T) {
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/slow.com", func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"status": "ERROR", "message": "Rate limit exceeded", "code": "RATE_LIMIT_EXCEEDED"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "yes", "price": "9.73"}, "ttlRemaining": 0}`))
	})

	client := newTestPorkbunClient(t, mux)
	res, err := client.CheckDomain(context.Background(), "slow.com")
	if err != nil {
		t.Fatalf("expected the 429 to be retried once, got: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("expected exactly 2 calls (initial + retry), got %d", calls)
	}
	if !res.toAvailability("slow.com").Registrable {
		t.Errorf("expected the retried check to succeed, got %+v", res)
	}
}

func TestPorkbunCreateDomainSuccess(t *testing.T) {
	var gotCost struct {
		Cost         int64  `json:"cost"`
		AgreeToTerms string `json:"agreeToTerms"`
		DryRun       bool   `json:"dryRun"`
	}
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/create/example.com", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotCost)
		_, _ = w.Write([]byte(`{
			"status": "SUCCESS",
			"domain": "example.com",
			"cost": 973,
			"orderId": 12345678,
			"balance": 4027,
			"requestId": "019e04fa-258d-7d11-aa86-4d5795c3fe8f"
		}`))
	})

	client := newTestPorkbunClient(t, mux)
	res, err := client.CreateDomain(context.Background(), "example.com", 973, false)
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}

	if gotPath != "/domain/create/example.com" {
		t.Errorf("unexpected request path: %q", gotPath)
	}
	if gotCost.Cost != 973 || gotCost.AgreeToTerms != "yes" || gotCost.DryRun {
		t.Errorf("unexpected create payload: %+v", gotCost)
	}
	if res.OrderID != 12345678 {
		t.Errorf("expected orderId 12345678, got %d", res.OrderID)
	}
	if res.DryRun {
		t.Error("expected a non-dry-run response")
	}
}

func TestPorkbunCreateDomainDryRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/create/example.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"status": "SUCCESS",
			"dryRun": true,
			"wouldSucceed": true,
			"operation": "registration",
			"domain": "example.com",
			"cost": 973,
			"costDisplay": "$9.73",
			"balance": 5000,
			"sufficientFunds": true,
			"message": "Dry run: this registration would succeed and cost $9.73."
		}`))
	})

	client := newTestPorkbunClient(t, mux)
	res, err := client.CreateDomain(context.Background(), "example.com", 973, true)
	if err != nil {
		t.Fatalf("CreateDomain(dryRun) failed: %v", err)
	}
	if !res.DryRun || !res.WouldSucceed {
		t.Errorf("expected a successful dry-run preview, got %+v", res)
	}
}

func TestPorkbunCreateDomainError(t *testing.T) {
	var createCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/create/poor.com", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&createCalls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": "ERROR", "message": "Insufficient funds.", "code": "INSUFFICIENT_FUNDS", "next_action": {"type": "add_funds", "hint": "Top up your account credit."}}`))
	})

	client := newTestPorkbunClient(t, mux)
	_, err := client.CreateDomain(context.Background(), "poor.com", 973, false)
	if err == nil {
		t.Fatal("expected an error for insufficient funds")
	}
	if !strings.Contains(err.Error(), "Insufficient funds") || !strings.Contains(err.Error(), "Top up") {
		t.Errorf("expected actionable error, got: %v", err)
	}
	if atomic.LoadInt32(&createCalls) != 1 {
		t.Errorf("create is a billable call and must not be retried on 400, got %d calls", createCalls)
	}
}

func TestPorkbunUpdateAutoRenew(t *testing.T) {
	for _, tc := range []struct {
		enabled    bool
		wantStatus string
	}{
		{true, "on"},
		{false, "off"},
	} {
		var got struct {
			Status string `json:"status"`
		}
		var gotPath string
		mux := http.NewServeMux()
		mux.HandleFunc("POST /domain/updateAutoRenew/example.com", func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = w.Write([]byte(`{"status": "SUCCESS"}`))
		})

		client := newTestPorkbunClient(t, mux)
		if err := client.UpdateAutoRenew(context.Background(), "example.com", tc.enabled); err != nil {
			t.Fatalf("UpdateAutoRenew(%v) failed: %v", tc.enabled, err)
		}
		if gotPath != "/domain/updateAutoRenew/example.com" {
			t.Errorf("unexpected request path: %q", gotPath)
		}
		if got.Status != tc.wantStatus {
			t.Errorf("expected status %q, got %q", tc.wantStatus, got.Status)
		}
	}
}

func TestPorkbunGetDomain(t *testing.T) {
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /domain/get/example.com", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{
			"status": "SUCCESS",
			"domain": {
				"domain": "example.com",
				"status": "ACTIVE",
				"tld": "com",
				"createDate": "2021-01-15 10:00:00",
				"expireDate": "2027-01-15 10:00:00",
				"securityLock": 1,
				"whoisPrivacy": 0,
				"autoRenew": 1,
				"apiAccess": 1,
				"notLocal": 0
			}
		}`))
	})

	client := newTestPorkbunClient(t, mux)
	dm, err := client.GetDomain(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("GetDomain failed: %v", err)
	}
	if gotPath != "/domain/get/example.com" {
		t.Errorf("unexpected request path: %q", gotPath)
	}

	if dm.Name != "example.com" || dm.Provider != "porkbun" {
		t.Errorf("unexpected domain identity: %+v", dm)
	}
	if !dm.AutoRenewal || !dm.Locked || dm.Privacy {
		t.Errorf("unexpected settings mapping: autoRenew=%v locked=%v privacy=%v", dm.AutoRenewal, dm.Locked, dm.Privacy)
	}
	if dm.ExpiryDate.Year() != 2027 || dm.ExpiryDate.Month() != time.January {
		t.Errorf("unexpected expiry parse: %v", dm.ExpiryDate)
	}
}

func TestPorkbunGetDomainMixedTypeFlags(t *testing.T) {
	// Live observation: the domain/get payload mixes numbers and numeric
	// strings for the flag fields (securityLock: 0 but autoRenew: "1").
	mux := http.NewServeMux()
	mux.HandleFunc("GET /domain/get/mixed.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"status": "SUCCESS",
			"domain": {
				"domain": "mixed.com",
				"status": "ACTIVE",
				"expireDate": "2027-09-02 17:23:20",
				"securityLock": 0,
				"whoisPrivacy": "1",
				"autoRenew": "1",
				"apiAccess": "1",
				"notLocal": 0
			}
		}`))
	})

	client := newTestPorkbunClient(t, mux)
	dm, err := client.GetDomain(context.Background(), "mixed.com")
	if err != nil {
		t.Fatalf("GetDomain with string-typed flags failed: %v", err)
	}
	if !dm.AutoRenewal || dm.Locked || !dm.Privacy {
		t.Errorf("unexpected flag mapping: autoRenew=%v locked=%v privacy=%v", dm.AutoRenewal, dm.Locked, dm.Privacy)
	}
}

func TestPorkbunGetDomainNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /domain/get/ghost.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": "ERROR", "message": "Domain not found.", "code": "NOT_FOUND"}`))
	})

	client := newTestPorkbunClient(t, mux)
	if _, err := client.GetDomain(context.Background(), "ghost.com"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected a not-found error, got: %v", err)
	}
}

func TestPorkbunPriceConversions(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"9.73", 973, false},
		{"11", 1100, false},
		{"0.99", 99, false},
		{" 10.115 ", 1012, false}, // rounded, not truncated
		{"abc", 0, true},
	} {
		got, err := priceToPennies(tc.in)
		if tc.wantErr && err == nil {
			t.Errorf("priceToPennies(%q): expected error", tc.in)
			continue
		}
		if !tc.wantErr && (err != nil || got != tc.want) {
			t.Errorf("priceToPennies(%q) = %d, %v; want %d pennies", tc.in, got, err, tc.want)
		}
	}
}

func TestPorkbunClientRequiresCredentials(t *testing.T) {
	client := NewPorkbunRegistrarClient("", "")
	if _, err := client.CheckDomain(context.Background(), "example.com"); err == nil || !strings.Contains(err.Error(), "requires api_key") {
		t.Errorf("expected a credentials error, got: %v", err)
	}
}

// --- Provider-level Purchaser capability ---

func TestPorkbunProviderCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/one.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "yes", "price": "9.73", "additional": {"renewal": {"price": "11.00"}}}, "ttlRemaining": 0}`))
	})
	mux.HandleFunc("POST /domain/checkDomain/two.dev", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "no"}, "ttlRemaining": 0}`))
	})

	provider := newTestPorkbunProvider(t, mux)
	results, err := provider.Check(context.Background(), []string{"one.com", "two.dev"})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Registrable || results[0].RegistrationCost != 9.73 || results[0].RenewalCost != 11.00 {
		t.Errorf("unexpected first result: %+v", results[0])
	}
	if results[1].Registrable {
		t.Errorf("unexpected second result: %+v", results[1])
	}
}

func TestPorkbunProviderCheckHonorsRateLimitWindow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/paced.com", func(w http.ResponseWriter, r *http.Request) {
		// ttlRemaining=10 would pace the NEXT check for ~11s; the canceled
		// context must cut the wait short.
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "yes", "price": "9.73"}, "ttlRemaining": 10}`))
	})

	provider := newTestPorkbunProvider(t, mux)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if _, err := provider.Check(ctx, []string{"paced.com", "paced.com"}); err == nil {
		t.Fatal("expected the canceled context to interrupt pacing between checks")
	}
}

func TestPorkbunProviderRegister(t *testing.T) {
	var createCost struct {
		Cost         int64  `json:"cost"`
		AgreeToTerms string `json:"agreeToTerms"`
	}
	var checkCalls, createCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/new.com", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&checkCalls, 1)
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "yes", "price": "9.73", "premium": "no"}, "ttlRemaining": 0}`))
	})
	mux.HandleFunc("POST /domain/create/new.com", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&createCalls, 1)
		_ = json.NewDecoder(r.Body).Decode(&createCost)
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "domain": "new.com", "cost": 973, "orderId": 42}`))
	})

	provider := newTestPorkbunProvider(t, mux)
	result, err := provider.Register(context.Background(), "new.com", nil)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if atomic.LoadInt32(&createCalls) != 1 {
		t.Errorf("expected exactly 1 create call, got %d", createCalls)
	}
	if createCost.Cost != 973 {
		t.Errorf("expected the create cost to be the checked price in pennies (973), got %d", createCost.Cost)
	}
	if createCost.AgreeToTerms != "yes" {
		t.Errorf("expected agreeToTerms=yes, got %q", createCost.AgreeToTerms)
	}
	if result.State != domains.RegistrationStateSucceeded || !result.Completed || result.DomainName != "new.com" {
		t.Errorf("unexpected registration result: %+v", result)
	}
}

func TestPorkbunProviderRegisterRefusesContact(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("no API call expected when a contact is supplied")
	})
	mux.HandleFunc("POST /domain/create/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("no create call expected when a contact is supplied")
	})

	provider := newTestPorkbunProvider(t, mux)
	contact := &domains.RegistrantContact{Email: "jane@example.com"}
	_, err := provider.Register(context.Background(), "new.com", contact)
	if err == nil || !strings.Contains(err.Error(), "contact") {
		t.Errorf("expected a contact-not-supported error, got: %v", err)
	}
}

func TestPorkbunProviderRegisterRefusesPremiumAndUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /domain/checkDomain/premium.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "yes", "premium": "yes", "price": "2999.00"}, "ttlRemaining": 0}`))
	})
	mux.HandleFunc("POST /domain/checkDomain/gone.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "response": {"avail": "no"}, "ttlRemaining": 0}`))
	})
	mux.HandleFunc("POST /domain/create/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("no create call expected for premium or unavailable domains")
	})

	provider := newTestPorkbunProvider(t, mux)
	if _, err := provider.Register(context.Background(), "premium.com", nil); err == nil || !strings.Contains(err.Error(), "premium") {
		t.Errorf("expected a premium refusal, got: %v", err)
	}
	if _, err := provider.Register(context.Background(), "gone.com", nil); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Errorf("expected an unavailable refusal, got: %v", err)
	}
}

func TestPorkbunProviderRegistrationStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /domain/get/live.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status": "SUCCESS", "domain": {"domain": "live.com", "expireDate": "2027-01-15 10:00:00", "autoRenew": 1}}`))
	})
	mux.HandleFunc("GET /domain/get/ghost.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": "ERROR", "message": "Domain not found."}`))
	})

	provider := newTestPorkbunProvider(t, mux)

	result, err := provider.RegistrationStatus(context.Background(), "live.com")
	if err != nil {
		t.Fatalf("RegistrationStatus(live) failed: %v", err)
	}
	if result.State != domains.RegistrationStateSucceeded || !result.Completed {
		t.Errorf("unexpected live status: %+v", result)
	}

	result, err = provider.RegistrationStatus(context.Background(), "ghost.com")
	if err != nil {
		t.Fatalf("RegistrationStatus(ghost) failed: %v", err)
	}
	if result.State != domains.RegistrationStateInProgress || result.Completed {
		t.Errorf("unexpected ghost status: %+v", result)
	}
}
