package output

import (
	"fmt"
	"strings"
	"time"
)

// Common formatters that can be reused across resource types

// Time formatters
var (
	// RelativeTimeFormatter formats time as relative duration (e.g., "2h", "3d")
	RelativeTimeFormatter = func(value interface{}) string {
		if t, ok := value.(time.Time); ok {
			return formatRelativeTime(time.Since(t))
		}
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return formatRelativeTime(time.Since(t))
			}
		}
		return "N/A"
	}

	// ExpiryTimeFormatter formats time until expiry (e.g., "30d", "-5d" for expired)
	ExpiryTimeFormatter = func(value interface{}) string {
		if t, ok := value.(time.Time); ok {
			duration := time.Until(t)
			if duration < 0 {
				return fmt.Sprintf("-%s", formatRelativeTime(-duration))
			}
			return formatRelativeTime(duration)
		}
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				duration := time.Until(t)
				if duration < 0 {
					return fmt.Sprintf("-%s", formatRelativeTime(-duration))
				}
				return formatRelativeTime(duration)
			}
		}
		return "N/A"
	}

	// AbsoluteTimeFormatter formats time as YYYY-MM-DD
	AbsoluteTimeFormatter = func(value interface{}) string {
		if t, ok := value.(time.Time); ok {
			return t.Format("2006-01-02")
		}
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t.Format("2006-01-02")
			}
		}
		return "N/A"
	}

	// DateTimeFormatter formats time as YYYY-MM-DD HH:MM:SS
	DateTimeFormatter = func(value interface{}) string {
		if t, ok := value.(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t.Format("2006-01-02 15:04:05")
			}
		}
		return "N/A"
	}
)

// Status formatters with color support
var (
	// StatusFormatter adds color coding based on common status values
	StatusFormatter = func(value interface{}) string {
		status := fmt.Sprintf("%v", value)
		return colorizeStatus(status)
	}

	// PlainStatusFormatter returns status without color
	PlainStatusFormatter = func(value interface{}) string {
		return fmt.Sprintf("%v", value)
	}
)

// Boolean formatters
var (
	// YesNoFormatter converts boolean to Yes/No
	YesNoFormatter = func(value interface{}) string {
		if b, ok := value.(bool); ok {
			if b {
				return "Yes"
			}
			return "No"
		}
		return "N/A"
	}

	// EnabledDisabledFormatter converts boolean to Enabled/Disabled
	EnabledDisabledFormatter = func(value interface{}) string {
		if b, ok := value.(bool); ok {
			if b {
				return "Enabled"
			}
			return "Disabled"
		}
		return "N/A"
	}

	// OnOffFormatter converts boolean to On/Off
	OnOffFormatter = func(value interface{}) string {
		if b, ok := value.(bool); ok {
			if b {
				return "On"
			}
			return "Off"
		}
		return "N/A"
	}

	// CheckMarkFormatter converts boolean to ✓/✗
	CheckMarkFormatter = func(value interface{}) string {
		if b, ok := value.(bool); ok {
			if b {
				return "✓"
			}
			return "✗"
		}
		return "N/A"
	}
)

// List and array formatters
var (
	// StringListFormatter joins string arrays with commas
	StringListFormatter = func(value interface{}) string {
		if list, ok := value.([]string); ok {
			return strings.Join(list, ",")
		}
		return "N/A"
	}

	// StringListSpaceFormatter joins string arrays with spaces
	StringListSpaceFormatter = func(value interface{}) string {
		if list, ok := value.([]string); ok {
			return strings.Join(list, " ")
		}
		return "N/A"
	}

	// TruncatedListFormatter creates a formatter that truncates long lists
	TruncatedListFormatter = func(maxLength int) ColumnFormatter {
		return func(value interface{}) string {
			if list, ok := value.([]string); ok {
				joined := strings.Join(list, ",")
				if len(joined) > maxLength {
					return joined[:maxLength-3] + "..."
				}
				return joined
			}
			return "N/A"
		}
	}

	// ListCountFormatter shows count of items in list (e.g., "3 items")
	ListCountFormatter = func(value interface{}) string {
		if list, ok := value.([]string); ok {
			count := len(list)
			if count == 1 {
				return "1 item"
			}
			return fmt.Sprintf("%d items", count)
		}
		return "0 items"
	}
)

// Numeric formatters
var (
	// CurrencyFormatter formats float as currency
	CurrencyFormatter = func(value interface{}) string {
		switch v := value.(type) {
		case float64:
			return fmt.Sprintf("$%.2f", v)
		case float32:
			return fmt.Sprintf("$%.2f", float64(v))
		case int:
			return fmt.Sprintf("$%d.00", v)
		case int64:
			return fmt.Sprintf("$%d.00", v)
		}
		return "N/A"
	}

	// ByteSizeFormatter formats bytes in human-readable format
	ByteSizeFormatter = func(value interface{}) string {
		var bytes int64
		switch v := value.(type) {
		case int64:
			bytes = v
		case int:
			bytes = int64(v)
		case float64:
			bytes = int64(v)
		default:
			return "N/A"
		}
		return formatByteSize(bytes)
	}

	// PercentageFormatter formats float as percentage
	PercentageFormatter = func(value interface{}) string {
		switch v := value.(type) {
		case float64:
			return fmt.Sprintf("%.1f%%", v*100)
		case float32:
			return fmt.Sprintf("%.1f%%", float64(v)*100)
		}
		return "N/A"
	}
)

// String formatters
var (
	// UpperCaseFormatter converts string to uppercase
	UpperCaseFormatter = func(value interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%v", value))
	}

	// LowerCaseFormatter converts string to lowercase
	LowerCaseFormatter = func(value interface{}) string {
		return strings.ToLower(fmt.Sprintf("%v", value))
	}

	// TitleCaseFormatter converts string to title case
	TitleCaseFormatter = func(value interface{}) string {
		return strings.Title(strings.ToLower(fmt.Sprintf("%v", value)))
	}

	// TruncateFormatter creates a formatter that truncates at specified length
	TruncateFormatter = func(maxLength int) ColumnFormatter {
		return func(value interface{}) string {
			str := fmt.Sprintf("%v", value)
			if len(str) > maxLength {
				return str[:maxLength-3] + "..."
			}
			return str
		}
	}
)

// Helper functions

// formatRelativeTime converts duration to human-readable format
func formatRelativeTime(duration time.Duration) string {
	if duration < 0 {
		duration = -duration
	}

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	} else if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dw", int(duration.Hours()/(24*7)))
	} else if duration < 365*24*time.Hour {
		return fmt.Sprintf("%dmo", int(duration.Hours()/(24*30)))
	} else {
		return fmt.Sprintf("%dy", int(duration.Hours()/(24*365)))
	}
}

// colorizeStatus adds ANSI color codes based on status value
func colorizeStatus(status string) string {
	switch strings.ToLower(status) {
	case "healthy", "active", "running", "ok", "up", "online", "ready":
		return fmt.Sprintf("\033[32m%s\033[0m", status) // Green
	case "warning", "pending", "degraded", "slow":
		return fmt.Sprintf("\033[33m%s\033[0m", status) // Yellow
	case "critical", "failed", "error", "down", "offline", "unhealthy":
		return fmt.Sprintf("\033[31m%s\033[0m", status) // Red
	case "expired", "stopped", "terminated", "dead":
		return fmt.Sprintf("\033[91m%s\033[0m", status) // Bright red
	case "unknown", "n/a":
		return fmt.Sprintf("\033[90m%s\033[0m", status) // Gray
	default:
		return status // No color
	}
}

// formatByteSize converts bytes to human-readable format
func formatByteSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CreateCustomFormatter creates a formatter with custom format string
func CreateCustomFormatter(format string) ColumnFormatter {
	return func(value interface{}) string {
		return fmt.Sprintf(format, value)
	}
}

// ChainFormatters combines multiple formatters (applies them in order)
func ChainFormatters(formatters ...ColumnFormatter) ColumnFormatter {
	return func(value interface{}) string {
		result := value
		for _, formatter := range formatters {
			if result != nil {
				result = formatter(result)
			}
		}
		return fmt.Sprintf("%v", result)
	}
}
