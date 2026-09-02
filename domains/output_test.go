package domains

import "testing"

// Regression: rows built from ManagedDomain carry the typed DomainStatus
// value, so the summary's former .(string) assertion counted everything as
// zero ("2 domains total: 0 healthy, 0 warning, ...").
func TestDomainTableSummaryFuncCountsTypedStatus(t *testing.T) {
	rows := []map[string]interface{}{
		{"status": StatusWarning},
		{"status": StatusWarning},
		{"status": StatusHealthy},
		{"status": StatusExpired},
	}
	got := DomainTableConfig.SummaryFunc(rows)
	want := "4 domains total: 1 healthy, 2 warning, 0 critical, 1 expired"
	if got != want {
		t.Errorf("summary mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestDomainTableSummaryFuncAcceptsPlainStrings(t *testing.T) {
	rows := []map[string]interface{}{
		{"status": "healthy"},
		{"status": "CRITICAL"},
	}
	got := DomainTableConfig.SummaryFunc(rows)
	want := "2 domains total: 1 healthy, 0 warning, 1 critical, 0 expired"
	if got != want {
		t.Errorf("summary mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestDomainTableSummaryFuncIgnoresUnknownStatus(t *testing.T) {
	rows := []map[string]interface{}{
		{"status": "mystery"},
		{"status": 42},
	}
	got := DomainTableConfig.SummaryFunc(rows)
	want := "2 domains total: 0 healthy, 0 warning, 0 critical, 0 expired"
	if got != want {
		t.Errorf("summary mismatch:\n got: %s\nwant: %s", got, want)
	}
}
