package domains

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"indietool/cli/output"
)

// SearchTableConfig defines the table layout for domain search results
var SearchTableConfig = output.TableConfig{
	DefaultColumns: []output.Column{
		{
			Name:     "DOMAIN",
			JSONPath: "domain",
			Required: true,
		},
		{
			Name:      "STATUS",
			JSONPath:  "status",
			Formatter: SearchStatusFormatter,
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
			WideOnly:  true,
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
				case "taken", "registered":
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

// SearchTableOptions creates table options for domain search based on command flags
func SearchTableOptions(format output.OutputFormat, wide, noColor, noHeaders bool, w io.Writer) output.TableOptions {
	return output.TableOptions{
		Format:    format,
		Wide:      wide,
		NoHeaders: noHeaders,
		NoColor:   noColor,
		Writer:    w,
	}
}

// GetSearchTableConfig returns the appropriate table config for search results
func GetSearchTableConfig(useColors bool) output.TableConfig {
	config := SearchTableConfig

	// Use plain status formatter for tabwriter to avoid ANSI alignment issues
	if !useColors {
		// Create a copy and modify the status formatter
		defaultColumns := make([]output.Column, len(config.DefaultColumns))
		copy(defaultColumns, config.DefaultColumns)

		// Find and update the STATUS column to use plain formatter
		for i := range defaultColumns {
			if defaultColumns[i].Name == "STATUS" {
				defaultColumns[i].Formatter = PlainSearchStatusFormatter
				break
			}
		}

		config.DefaultColumns = defaultColumns
	}

	return config
}

// SortSearchResults sorts domain search results by availability first, then by domain name
func SortSearchResults(results []DomainSearchResult) {
	sort.Slice(results, func(i, j int) bool {
		// First sort by status priority (Available > Taken > Error)
		statusPriorityI := getSearchStatusPriority(results[i])
		statusPriorityJ := getSearchStatusPriority(results[j])

		if statusPriorityI != statusPriorityJ {
			return statusPriorityI < statusPriorityJ
		}

		// Then sort alphabetically by domain name
		return results[i].Domain < results[j].Domain
	})
}

// getSearchStatusPriority returns sorting priority for different status types
func getSearchStatusPriority(result DomainSearchResult) int {
	if result.Error != "" {
		return 3 // Errors last
	}
	if result.Available {
		return 1 // Available first
	}
	return 2 // Taken second
}

// ConvertSearchResultsToTableRows converts search results to table rows for rendering
func ConvertSearchResultsToTableRows(results []DomainSearchResult) []map[string]interface{} {
	// Sort results first
	SortSearchResults(results)

	rows := make([]map[string]interface{}, 0, len(results))

	for _, result := range results {
		tld := extractTLD(result.Domain)

		row := map[string]interface{}{
			"domain":      result.Domain,
			"status":      getSearchStatus(result),
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

// getSearchStatus determines the status string for a domain search result
func getSearchStatus(result DomainSearchResult) string {
	if result.Error != "" {
		return "Error"
	}
	if result.Available {
		return "Available"
	}
	return "Taken"
}

// Formatter functions for search table columns

// SearchStatusFormatter formats domain availability status with colors
func SearchStatusFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}

	status := fmt.Sprintf("%v", value)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "available":
		return fmt.Sprintf("\033[32m%s\033[0m", status) // Green
	case "taken", "registered":
		return fmt.Sprintf("\033[31m%s\033[0m", status) // Red
	case "error":
		return fmt.Sprintf("\033[33m%s\033[0m", status) // Yellow
	default:
		return status
	}
}

// PlainSearchStatusFormatter formats domain status without colors
func PlainSearchStatusFormatter(value interface{}) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%v", value)
}