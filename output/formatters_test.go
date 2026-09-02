package output

import (
	"testing"
	"time"
)

func TestRelativeTimeFormatterZeroTime(t *testing.T) {
	if got := RelativeTimeFormatter(time.Time{}); got != "N/A" {
		t.Errorf("zero time should render N/A, got %q", got)
	}
	if got := RelativeTimeFormatter("0001-01-01T00:00:00Z"); got != "N/A" {
		t.Errorf("zero RFC3339 string should render N/A, got %q", got)
	}
	recent := time.Now().Add(-2 * time.Hour)
	if got := RelativeTimeFormatter(recent); got == "N/A" || got == "" {
		t.Errorf("recent time should render a relative duration, got %q", got)
	}
}

func TestExpiryTimeFormatterZeroTime(t *testing.T) {
	if got := ExpiryTimeFormatter(time.Time{}); got != "N/A" {
		t.Errorf("zero expiry should render N/A, got %q", got)
	}
	if got := ExpiryTimeFormatter("0001-01-01T00:00:00Z"); got != "N/A" {
		t.Errorf("zero RFC3339 expiry string should render N/A, got %q", got)
	}
	future := time.Now().Add(30 * 24 * time.Hour)
	if got := ExpiryTimeFormatter(future); got == "N/A" || got == "" {
		t.Errorf("future expiry should render a duration, got %q", got)
	}
}
