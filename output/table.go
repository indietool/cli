package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
)

// NewTable creates a new table with the given configuration
func NewTable(config TableConfig, options ...TableOptions) *Table {
	opts := DefaultTableOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Determine which columns to use
	columns := make([]Column, len(config.DefaultColumns))
	copy(columns, config.DefaultColumns)

	// Add wide columns if wide format is requested
	if opts.Wide && opts.Format == FormatWide {
		columns = append(columns, config.WideColumns...)
	}

	return &Table{
		columns:     columns,
		rows:        make([]map[string]interface{}, 0),
		format:      opts.Format,
		showHeaders: !opts.NoHeaders,
		colorize:    !opts.NoColor,
		writer:      opts.Writer,
		config:      config,
	}
}

// Fluent API methods for configuration

// WithFormat sets the output format
func (t *Table) WithFormat(format OutputFormat) *Table {
	t.format = format
	// If switching to wide format, add wide columns
	if format == FormatWide {
		t.enableWideColumns()
	}
	return t
}

// WithWriter sets the output writer
func (t *Table) WithWriter(w io.Writer) *Table {
	t.writer = w
	return t
}

// WithColorize enables or disables color output
func (t *Table) WithColorize(colorize bool) *Table {
	t.colorize = colorize
	return t
}

// WithHeaders enables or disables column headers
func (t *Table) WithHeaders(show bool) *Table {
	t.showHeaders = show
	return t
}

// EnableWideFormat adds wide-only columns to the table
func (t *Table) EnableWideFormat() *Table {
	t.enableWideColumns()
	t.format = FormatWide
	return t
}

// enableWideColumns adds wide-only columns if not already present
func (t *Table) enableWideColumns() {
	// Check if wide columns are already added
	hasWideColumns := false
	for _, col := range t.columns {
		if col.WideOnly {
			hasWideColumns = true
			break
		}
	}

	// Add wide columns if not present
	if !hasWideColumns {
		t.columns = append(t.columns, t.config.WideColumns...)
	}
}

// Data manipulation methods

// AddRow adds a single data row (converts struct to map using reflection)
func (t *Table) AddRow(data interface{}) *Table {
	if rowMap := convertToMap(data); rowMap != nil {
		t.rows = append(t.rows, rowMap)
	}
	return t
}

// AddRows adds multiple data rows
func (t *Table) AddRows(data interface{}) *Table {
	if slice := convertSliceToMaps(data); slice != nil {
		t.rows = append(t.rows, slice...)
	}
	return t
}

// SetRows replaces all rows with new data
func (t *Table) SetRows(data interface{}) *Table {
	t.rows = make([]map[string]interface{}, 0)
	return t.AddRows(data)
}

// ClearRows removes all data rows
func (t *Table) ClearRows() *Table {
	t.rows = make([]map[string]interface{}, 0)
	return t
}

// Rendering methods

// Render outputs the table in the configured format
func (t *Table) Render() error {
	switch t.format {
	case FormatTable, FormatWide:
		return t.renderTable()
	case FormatJSON:
		return t.renderJSON()
	case FormatYAML:
		return t.renderYAML()
	default:
		return fmt.Errorf("unsupported format: %s", t.format)
	}
}

// RenderWithSummary renders table with optional summary footer
func (t *Table) RenderWithSummary() error {
	if err := t.Render(); err != nil {
		return err
	}

	// Add summary if available and format supports it
	if t.config.SummaryFunc != nil && (t.format == FormatTable || t.format == FormatWide) {
		summary := t.config.SummaryFunc(t.rows)
		if summary != "" {
			fmt.Fprintf(t.writer, "\n%s\n", summary)
		}
	}

	return nil
}

// Table rendering implementation

func (t *Table) renderTable() error {
	if len(t.rows) == 0 {
		fmt.Fprintf(t.writer, "No data available\n")
		return nil
	}

	// Create tabwriter with appropriate settings
	// minwidth, tabwidth, padding, padchar, flags
	w := tabwriter.NewWriter(t.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Render headers
	if t.showHeaders {
		t.renderHeaders(w)
	}

	// Render data rows
	for _, row := range t.rows {
		t.renderRow(w, row)
	}

	return nil
}

func (t *Table) renderHeaders(w *tabwriter.Writer) {
	parts := make([]string, len(t.columns))
	for i, col := range t.columns {
		parts[i] = col.Name
	}
	fmt.Fprintf(w, "%s\n", strings.Join(parts, "\t"))
}

func (t *Table) renderRow(w *tabwriter.Writer, row map[string]interface{}) {
	parts := make([]string, len(t.columns))
	for i, col := range t.columns {
		value := t.formatCellValue(row, col)
		if col.Truncate {
			truncateAt := col.TruncateAt
			if truncateAt == 0 {
				truncateAt = col.Width - 3 // Use column width for truncation
				if truncateAt <= 0 {
					truncateAt = 30 // Default truncation
				}
			}
			value = t.truncateText(value, truncateAt)
		}
		parts[i] = value
	}
	fmt.Fprintf(w, "%s\n", strings.Join(parts, "\t"))
}

func (t *Table) formatCellValue(row map[string]interface{}, col Column) string {
	// Get value using JSON path (supports nested paths like "cost.renewal_price")
	value := getValueByPath(row, col.JSONPath)

	// Apply formatter if available
	if col.Formatter != nil {
		return col.Formatter(value)
	}

	// Default string conversion
	if value == nil {
		return "N/A"
	}
	return fmt.Sprintf("%v", value)
}

func (t *Table) truncateText(text string, maxLen int) string {
	// Simple truncation - tabwriter handles the spacing
	if len(text) <= maxLen {
		return text
	}
	
	// Remove ANSI color codes for length calculation
	cleanText := removeANSIColors(text)
	if len(cleanText) <= maxLen {
		return text
	}
	
	// Truncate and add ellipsis
	runes := []rune(cleanText)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return text
}

// JSON rendering

func (t *Table) renderJSON() error {
	encoder := json.NewEncoder(t.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(t.rows)
}

// YAML rendering

func (t *Table) renderYAML() error {
	encoder := yaml.NewEncoder(t.writer)
	defer encoder.Close()
	return encoder.Encode(t.rows)
}

// Utility functions

// convertToMap converts a struct to map[string]interface{} using reflection
func convertToMap(data interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get field name from json tag or use field name
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Skip fields marked as "-"
		if fieldName == "-" {
			continue
		}

		result[fieldName] = fieldValue.Interface()
	}

	return result
}

// convertSliceToMaps converts a slice of structs to []map[string]interface{}
func convertSliceToMaps(data interface{}) []map[string]interface{} {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return nil
	}

	result := make([]map[string]interface{}, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		if rowMap := convertToMap(v.Index(i).Interface()); rowMap != nil {
			result = append(result, rowMap)
		}
	}

	return result
}

// getValueByPath retrieves a value from a map using a dot-separated path
func getValueByPath(data map[string]interface{}, path string) interface{} {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if current == nil {
			return nil
		}

		value, exists := current[part]
		if !exists {
			return nil
		}

		// If this is the last part, return the value
		if i == len(parts)-1 {
			return value
		}

		// Otherwise, the value should be a map for the next iteration
		if nextMap, ok := value.(map[string]interface{}); ok {
			current = nextMap
		} else {
			return nil
		}
	}

	return nil
}

// removeANSIColors removes ANSI color codes from a string
func removeANSIColors(s string) string {
	// Simple regex-free approach to remove ANSI escape sequences
	var result strings.Builder
	inEscape := false

	for _, r := range s {
		if r == '\033' { // ESC character
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' { // End of color sequence
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// Helper functions for common table operations

// QuickTable is a convenience function for simple table rendering
func QuickTable(data interface{}, columns []Column, options ...TableOptions) error {
	config := TableConfig{
		DefaultColumns: columns,
	}

	table := NewTable(config, options...)
	table.AddRows(data)
	return table.Render()
}

// QuickTableWithSummary is like QuickTable but includes a summary
func QuickTableWithSummary(data interface{}, columns []Column, summaryFunc SummaryFormatter, options ...TableOptions) error {
	config := TableConfig{
		DefaultColumns: columns,
		SummaryFunc:    summaryFunc,
	}

	table := NewTable(config, options...)
	table.AddRows(data)
	return table.RenderWithSummary()
}
