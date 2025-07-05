package output

import (
	"io"
	"os"
)

// OutputFormat defines the output format type
type OutputFormat string

const (
	FormatTable  OutputFormat = "table"
	FormatWide   OutputFormat = "wide"
	FormatJSON   OutputFormat = "json"
	FormatYAML   OutputFormat = "yaml"
	FormatCustom OutputFormat = "custom"
)

// Note: Column alignment is handled automatically by text/tabwriter
// No manual alignment configuration needed

// ColumnFormatter transforms a value for display
type ColumnFormatter func(value interface{}) string

// Column defines a table column configuration
type Column struct {
	Name       string          // Display name (e.g., "NAME", "STATUS")
	JSONPath   string          // JSON path for custom output (e.g., "name", "status")
	Width      int             // Fixed width for truncation (0 = no truncation)
	Formatter  ColumnFormatter // Custom formatter function
	Truncate   bool            // Whether to truncate long values
	TruncateAt int             // Truncate threshold (0 = use width-3)
	Required   bool            // Always show this column
	WideOnly   bool            // Only show in wide format
}

// SummaryFormatter generates a summary line from table data
type SummaryFormatter func(rows []map[string]interface{}) string

// TableConfig defines the complete table configuration for a resource type
type TableConfig struct {
	DefaultColumns []Column         // Standard table view columns
	WideColumns    []Column         // Additional columns for wide view
	Formatters     map[string]ColumnFormatter // Named formatters for reuse
	SummaryFunc    SummaryFormatter // Optional summary generator
}

// Table represents a configured output table
type Table struct {
	columns     []Column                     // Active columns for this table
	rows        []map[string]interface{}     // Data rows
	format      OutputFormat                 // Output format
	showHeaders bool                         // Whether to show column headers
	colorize    bool                         // Whether to colorize output
	writer      io.Writer                    // Output destination
	config      TableConfig                  // Original table configuration
}

// TableOptions provides configuration options for table creation
type TableOptions struct {
	Format      OutputFormat
	Wide        bool
	NoHeaders   bool
	NoColor     bool
	Writer      io.Writer
	ShowSummary bool
}

// DefaultTableOptions returns sensible default options
func DefaultTableOptions() TableOptions {
	return TableOptions{
		Format:      FormatTable,
		Wide:        false,
		NoHeaders:   false,
		NoColor:     false,
		Writer:      os.Stdout,
		ShowSummary: false,
	}
}
