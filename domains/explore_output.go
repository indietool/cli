package domains

import (
	"fmt"
	"indietool/cli/output"
	"io"
	"sort"
	"strings"
	"time"
)

// ExploreTableConfig defines the table layout for domain exploration results
var ExploreTableConfig = output.TableConfig{
	DefaultColumns: []output.Column{
		{
			Name:     "DOMAIN",
			JSONPath: "domain",
			Required: true,
		},
		{
			Name:      "STATUS",
			JSONPath:  "status",
			Formatter: ExploreStatusFormatter,
			Required:  true,
		},
		{
			Name:     "TLD",
			JSONPath: "tld",
			Required: true,
		},
	},

	WideColumns: []output.Column{
		{
			Name:      "REGISTRAR",
			JSONPath:  "registrar",
			Formatter: DashIfEmptyFormatter,
			Required:  true,
		},
		{
			Name:      "COST",
			JSONPath:  "cost",
			Formatter: CostFormatter,
			WideOnly:  true,
		},
		{
			Name:      "EXPIRY",
			JSONPath:  "expiry_date",
			Formatter: ExpiryDateFormatter,
			WideOnly:  true,
		},
		{
			Name:      "ERROR",
			JSONPath:  "error",
			Formatter: DashIfEmptyFormatter,
			WideOnly:  true,
		},
	},

	SummaryFunc: func(rows []map[string]interface{}) string {
		total := len(rows)
		available, taken, errors := 0, 0, 0

		for _, row := range rows {
			if status, ok := row["status"].(string); ok {
				switch strings.ToLower(strings.TrimSpace(status)) {
				case "available":
					available++
				case "taken":
					taken++
				case "error":
					errors++
				}
			}
		}

		summary := fmt.Sprintf("%d domains checked: %d available, %d taken", total, available, taken)
		if errors > 0 {
			summary += fmt.Sprintf(", %d errors", errors)
		}

		return summary
	},
}

// ExploreTableOptions creates table options for domain exploration based on command flags
func ExploreTableOptions(format output.OutputFormat, wide, noColor, noHeaders bool, w io.Writer) output.TableOptions {
	return output.TableOptions{
		Format:    format,
		Wide:      wide,
		NoHeaders: noHeaders,
		NoColor:   noColor,
		Writer:    w,
	}
}

// GetExploreTableConfig returns the appropriate table config for exploration results
func GetExploreTableConfig(useColors bool) output.TableConfig {
	config := ExploreTableConfig

	// Use plain status formatter for tabwriter to avoid ANSI alignment issues
	if !useColors {
		// Create a copy and modify the status formatter
		defaultColumns := make([]output.Column, len(config.DefaultColumns))
		copy(defaultColumns, config.DefaultColumns)

		// Find and update the STATUS column to use plain formatter
		for i := range defaultColumns {
			if defaultColumns[i].Name == "STATUS" {
				defaultColumns[i].Formatter = PlainExploreStatusFormatter
				break
			}
		}

		config.DefaultColumns = defaultColumns
	}

	return config
}

// SortExploreResults sorts domain search results by availability first, then by domain name
func SortExploreResults(results []DomainSearchResult) {
	sort.Slice(results, func(i, j int) bool {
		// First sort by status priority (Available > Taken > Error)
		statusPriorityI := getStatusPriority(results[i])
		statusPriorityJ := getStatusPriority(results[j])

		if statusPriorityI != statusPriorityJ {
			return statusPriorityI < statusPriorityJ
		}

		// Then sort alphabetically by domain name
		return results[i].Domain < results[j].Domain
	})
}

// getStatusPriority returns sorting priority for different status types
func getStatusPriority(result DomainSearchResult) int {
	if result.Error != "" {
		return 3 // Errors last
	}
	if result.Available {
		return 1 // Available first
	}
	return 2 // Taken second
}

// ConvertExploreResultsToTableRows converts ExploreResult to table rows for rendering
func (er *ExploreResult) ConvertToTableRows() []map[string]interface{} {
	// Sort results first
	SortExploreResults(er.Results)

	rows := make([]map[string]interface{}, 0, len(er.Results))

	for _, result := range er.Results {
		tld := extractTLD(result.Domain)

		row := map[string]interface{}{
			"domain":      result.Domain,
			"status":      getExploreStatus(result),
			"tld":         tld,
			"registrar":   "",          // Not available in DomainSearchResult
			"cost":        0.0,         // Not available in DomainSearchResult
			"expiry_date": time.Time{}, // Not available in DomainSearchResult
			"error":       result.Error,
		}
		rows = append(rows, row)
	}

	return rows
}

// getExploreStatus determines the status string for a domain search result
func getExploreStatus(result DomainSearchResult) string {
	if result.Error != "" {
		return "Error"
	}
	if result.Available {
		return "Available"
	}
	return "Taken"
}

// extractTLD extracts the TLD from a domain name
func extractTLD(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}
	return parts[len(parts)-1]
}

// Formatter functions for explore table columns

// ExploreStatusFormatter formats domain availability status with colors
func ExploreStatusFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}

	status := fmt.Sprintf("%v", value)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "available":
		return fmt.Sprintf("\033[32m%s\033[0m", status) // Green
	case "taken":
		return fmt.Sprintf("\033[31m%s\033[0m", status) // Red
	case "error":
		return fmt.Sprintf("\033[33m%s\033[0m", status) // Yellow
	default:
		return status
	}
}

// PlainExploreStatusFormatter formats domain status without colors
func PlainExploreStatusFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%v", value)
}

// DashIfEmptyFormatter returns a dash if the value is empty
func DashIfEmptyFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}

	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	if str == "" {
		return "-"
	}
	return str
}

// CostFormatter formats cost values with currency symbol
func CostFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}

	switch v := value.(type) {
	case float64:
		if v == 0 {
			return "-"
		}
		return fmt.Sprintf("$%.2f", v)
	case float32:
		if v == 0 {
			return "-"
		}
		return fmt.Sprintf("$%.2f", v)
	case int:
		if v == 0 {
			return "-"
		}
		return fmt.Sprintf("$%d.00", v)
	default:
		str := strings.TrimSpace(fmt.Sprintf("%v", value))
		if str == "" || str == "0" {
			return "-"
		}
		return str
	}
}

// ExpiryDateFormatter formats expiry dates
func ExpiryDateFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}

	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			return "-"
		}
		return v.Format("2006-01-02")
	case string:
		if v == "" {
			return "-"
		}
		// Try to parse the string as a time
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Format("2006-01-02")
		}
		return v
	default:
		str := strings.TrimSpace(fmt.Sprintf("%v", value))
		if str == "" {
			return "-"
		}
		return str
	}
}

