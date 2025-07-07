package domains

import (
	"fmt"
	"io"
	"strings"

	"indietool/cli/output"
)

// DomainTableConfig defines the table layout for domain resources
var DomainTableConfig = output.TableConfig{
	DefaultColumns: []output.Column{
		{
			Name:     "NAME",
			JSONPath: "name",
			Required: true,
		},
		{
			Name:     "PROVIDER",
			JSONPath: "provider",
			Required: true,
		},
		{
			Name:      "STATUS",
			JSONPath:  "status",
			Formatter: output.StatusFormatter,
			Required:  true,
		},
		{
			Name:      "EXPIRES",
			JSONPath:  "expiry_date",
			Formatter: output.ExpiryTimeFormatter,
			Required:  true,
		},
		{
			Name:      "AUTO-RENEW",
			JSONPath:  "auto_renewal",
			Formatter: output.YesNoFormatter,
			Required:  true,
		},
		{
			Name:      "AGE",
			JSONPath:  "last_updated",
			Formatter: output.RelativeTimeFormatter,
			Required:  true,
		},
	},

	WideColumns: []output.Column{
		{
			Name:      "NAMESERVERS",
			JSONPath:  "nameservers",
			Width:     40,
			Formatter: output.TruncatedListFormatter(35),
			Truncate:  true,
			WideOnly:  true,
		},
		{
			Name:      "COST",
			JSONPath:  "cost.renewal_price",
			Formatter: output.CurrencyFormatter,
			WideOnly:  true,
		},
		{
			Name:      "UPDATED",
			JSONPath:  "last_updated",
			Formatter: output.RelativeTimeFormatter,
			WideOnly:  true,
		},
	},

	SummaryFunc: func(rows []map[string]interface{}) string {
		total := len(rows)
		healthy, warning, critical, expired := 0, 0, 0, 0

		for _, row := range rows {
			if status, ok := row["status"].(string); ok {
				switch DomainStatus(strings.ToLower(status)) {
				case StatusHealthy:
					healthy++
				case StatusWarning:
					warning++
				case StatusCritical:
					critical++
				case StatusExpired:
					expired++
				}
			}
		}

		summary := fmt.Sprintf("%d domains total: %d healthy, %d warning, %d critical, %d expired",
			total, healthy, warning, critical, expired)

		// Add last synced info if available
		// This could be enhanced to include sync timestamp from the result
		return summary
	},
}

// DomainTableOptions creates table options based on command flags
func DomainTableOptions(format output.OutputFormat, wide, noColor, noHeaders bool, w io.Writer) output.TableOptions {
	return output.TableOptions{
		Format:    format,
		Wide:      wide,
		NoHeaders: noHeaders,
		NoColor:   noColor,
		Writer:    w,
	}
}

// GetOutputFormat determines the output format from command flags
func GetOutputFormat(jsonOutput, wideOutput bool) output.OutputFormat {
	if jsonOutput {
		return output.FormatJSON
	}
	if wideOutput {
		return output.FormatWide
	}
	return output.FormatTable
}

// GetDomainTableConfig returns the appropriate table config based on whether colors should be used
// Colors are disabled for tabwriter-based formats (table/wide) to prevent ANSI codes from
// breaking column alignment in text/tabwriter
func GetDomainTableConfig(useColors bool) output.TableConfig {
	config := DomainTableConfig

	// Use plain status formatter for tabwriter to avoid ANSI alignment issues
	if !useColors {
		// Create a copy and modify the status formatter
		defaultColumns := make([]output.Column, len(config.DefaultColumns))
		copy(defaultColumns, config.DefaultColumns)

		// Find and update the STATUS column to use plain formatter
		for i := range defaultColumns {
			if defaultColumns[i].Name == "STATUS" {
				defaultColumns[i].Formatter = output.PlainStatusFormatter
				break
			}
		}

		config.DefaultColumns = defaultColumns
	}

	return config
}
