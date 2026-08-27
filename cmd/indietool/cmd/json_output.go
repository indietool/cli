package cmd

import (
	"encoding/json"
	"os"
)

// printJSON writes v as indented JSON to stdout.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
