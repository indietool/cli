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
	}

	if len(args) > 0 {
		data["args"] = args
	}
	if duration > 0 {
		data["duration_ms"] = duration.Milliseconds()
	}

	payload := &UmamiPayload{
		Type: "event",
	}

	payload.Payload.Hostname = "cli.indietool.dev"
	payload.Payload.Language = getSystemLanguage()
	payload.Payload.Referrer = ""
	payload.Payload.Title = command
	payload.Payload.URL = commandToURL(command)
	payload.Payload.Screen = getScreenSize()
	payload.Payload.Data = data

	return payload
}
