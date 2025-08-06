package metrics

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"
)

// UmamiPayload represents the complete payload for Umami API
type UmamiPayload struct {
	Type      string `json:"type"`
	UserAgent string `json:"-"` // Used for HTTP header, not sent in JSON
	Payload   struct {
		Hostname string                 `json:"hostname"`
		Language string                 `json:"language"`
		Referrer string                 `json:"referrer"`
		Title    string                 `json:"title"`
		URL      string                 `json:"url"`
		Website  string                 `json:"website"`
		Name     string                 `json:"name"`
		Screen   string                 `json:"screen"`
		Tag      string                 `json:"tag"`
		Data     map[string]interface{} `json:"data,omitempty"`
	} `json:"payload"`
}

// commandToURL converts command name to URL format
func commandToURL(command string) string {
	if command == "" {
		return "/"
	}
	return "https://cli.indietool.dev/" + strings.ReplaceAll(command, " ", "/")
}

// getSystemLanguage attempts to derive language from system environment
func getSystemLanguage() string {
	// Try various environment variables for locale
	for _, envVar := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if lang := os.Getenv(envVar); lang != "" {
			// Extract language part (before dot or underscore)
			if idx := strings.IndexAny(lang, "._"); idx > 0 {
				lang = lang[:idx]
			}
			// Convert to standard format (e.g., en_US -> en-US)
			lang = strings.ReplaceAll(lang, "_", "-")
			if len(lang) >= 2 {
				return lang
			}
		}
	}
	// Default fallback
	return "en-US"
}

// getScreenSize attempts to get terminal dimensions
func getScreenSize() string {
	// Try to get terminal size from stdout
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return fmt.Sprintf("%dx%d", width, height)
	}
	// Try stderr as fallback
	if width, height, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
		return fmt.Sprintf("%dx%d", width, height)
	}
	// Default fallback
	return "80x24"
}

// NewCommandEvent creates a tracking event for command execution
func NewCommandEvent(command string, args []string, duration time.Duration) *UmamiPayload {
	data := map[string]interface{}{
		"os":   runtime.GOOS,
		"arch": runtime.GOARCH,
		"lang": getSystemLanguage(),
	}

	if duration > 0 {
		data["duration_ms"] = duration.Milliseconds()
	}

	payload := &UmamiPayload{
		Type: "event",
	}

	payload.Payload.Hostname = "cli.indietool.dev"
	payload.Payload.Language = data["lang"].(string)
	payload.Payload.Referrer = ""
	payload.Payload.Title = command
	payload.Payload.Name = command
	payload.Payload.URL = commandToURL(command)
	payload.Payload.Screen = getScreenSize()
	payload.Payload.Data = data

	return payload
}

// Sanitise removes sensitive information from the payload while preserving useful metadata
func (p *UmamiPayload) Sanitise() {
	if args, ok := p.Payload.Data["args"].([]string); ok {
		sanitizedArgs, extra := sanitizeArgs(p.Payload.Name, args)
		p.Payload.Data["args"] = sanitizedArgs

		// Add extra metadata
		for k, v := range extra {
			p.Payload.Data[k] = v
		}
	}
}

// sanitizeArgs removes sensitive values from command arguments while preserving useful metadata
func sanitizeArgs(command string, args []string) ([]string, map[string]interface{}) {
	extra := make(map[string]interface{})

	// Handle different command patterns
	switch {
	case strings.HasPrefix(command, "secrets"):
		return sanitizeSecretsArgs(args, extra)
	case strings.HasPrefix(command, "dns"):
		return sanitizeDNSArgs(args, extra)
	case strings.HasPrefix(command, "config"):
		return sanitizeConfigArgs(args, extra)
	case strings.HasPrefix(command, "domain"):
		return sanitizeDomainArgs(args, extra)
	default:
		return sanitizeGenericArgs(args), extra
	}
}

// sanitizeSecretsArgs handles secrets command arguments
func sanitizeSecretsArgs(args []string, extra map[string]interface{}) ([]string, map[string]interface{}) {
	var sanitized []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			// Keep flags
			sanitized = append(sanitized, arg)
		} else if strings.Contains(arg, "@") {
			// Handle @database syntax - track usage but redact database name
			extra["uses_custom_db"] = true
			sanitized = append(sanitized, "<redacted>@<database>")
		} else {
			// Replace sensitive values with placeholders
			switch {
			case len(sanitized) == 0: // First non-flag arg is usually the subcommand
				sanitized = append(sanitized, arg)
			default:
				sanitized = append(sanitized, "<redacted>")
			}
		}
	}

	return sanitized, extra
}

// sanitizeDNSArgs handles DNS command arguments
func sanitizeDNSArgs(args []string, extra map[string]interface{}) ([]string, map[string]interface{}) {
	var sanitized []string
	skipNext := false

	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(arg, "-") {
			sanitized = append(sanitized, arg)

			// Track provider usage without revealing which provider
			if arg == "--provider" || arg == "-p" {
				extra["uses_provider_flag"] = true
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					skipNext = true
				}
			}
		} else {
			// Keep subcommand, redact domain names and values
			switch {
			case len(sanitized) == 0:
				sanitized = append(sanitized, arg)
			default:
				sanitized = append(sanitized, "<redacted>")
			}
		}
	}

	return sanitized, extra
}

// sanitizeConfigArgs handles config command arguments
func sanitizeConfigArgs(args []string, extra map[string]interface{}) ([]string, map[string]interface{}) {
	var sanitized []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			sanitized = append(sanitized, arg)
		} else {
			// Keep subcommands like "add", "provider", but redact values
			if len(sanitized) < 3 { // Allow "add provider cloudflare" but redact after
				sanitized = append(sanitized, arg)
			} else {
				sanitized = append(sanitized, "<redacted>")
			}
		}
	}

	return sanitized, extra
}

// sanitizeDomainArgs handles domain command arguments
func sanitizeDomainArgs(args []string, extra map[string]interface{}) ([]string, map[string]interface{}) {
	var sanitized []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			sanitized = append(sanitized, arg)
		} else {
			// Keep subcommand, redact domain names
			if len(sanitized) == 0 {
				sanitized = append(sanitized, arg)
			} else {
				sanitized = append(sanitized, "<redacted>")
			}
		}
	}

	return sanitized, extra
}

// sanitizeGenericArgs provides basic sanitization for unknown commands
func sanitizeGenericArgs(args []string) []string {
	var sanitized []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			sanitized = append(sanitized, arg)
		} else {
			// Keep first non-flag arg (usually subcommand), redact others
			if len(sanitized) == 0 {
				sanitized = append(sanitized, arg)
			} else {
				sanitized = append(sanitized, "<redacted>")
			}
		}
	}

	return sanitized
}
