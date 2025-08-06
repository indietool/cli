package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Agent handles sending metrics to Umami
type Agent struct {
	config *Config
	client *http.Client
}

// NewAgent creates a new metrics agent
func NewAgent() *Agent {
	return &Agent{
		config: NewConfig(),
		client: &http.Client{
			Timeout: 5 * time.Second, // Quick timeout to avoid blocking
		},
	}
}

func (a *Agent) SetVersion(version string) {
	a.config.SetVersion(version)
}

// Observe sends a command execution event asynchronously and returns a channel to wait on
func (a *Agent) Observe(command string, args []string, metadata map[string]string, duration time.Duration) <-chan struct{} {
	done := make(chan struct{})

	if !a.config.Enabled {
		close(done)
		return done
	}

	// Run in background goroutine to avoid blocking
	go func() {
		defer close(done)
		event := NewCommandEvent(command, args, duration)

		// Add metadata to event
		if metadata != nil {
			for k, v := range metadata {
				event.Payload.Data[k] = v
			}
		}

		event.Sanitise() // Remove sensitive information
		a.sendEvent(event)
	}()

	return done
}

// sendEvent sends the event to Umami API
func (a *Agent) sendEvent(event *UmamiPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	event.Payload.Website = a.config.WebsiteID
	event.Payload.Tag = a.config.Tag

	jsonData, err := json.Marshal(event)
	if err != nil {
		return // Silently fail - don't crash the CLI
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.config.UserAgent)

	// Send request and ignore response/errors to avoid blocking CLI
	a.client.Do(req)
}
