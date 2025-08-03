package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
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
func (a *Agent) Observe(command string, args []string, duration time.Duration) <-chan struct{} {
	done := make(chan struct{})

	if !a.config.Enabled {
		close(done)
		return done
	}

	// Run in background goroutine to avoid blocking
	go func() {
		defer close(done)
		event := NewCommandEvent(command, args, duration)
		a.sendEvent(event)
	}()

	return done
}

// sendEvent sends the event to Umami API
func (a *Agent) sendEvent(event *UmamiPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	event.Payload.Website = a.config.WebsiteID

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
	resp, rerr := a.client.Do(req)
	if rerr != nil {
		log.Errorf("failed to send metrics: %s", rerr)
	}
	log.Debugf("resp: %+v", resp)
}
