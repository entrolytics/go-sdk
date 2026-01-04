package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Config represents the configuration for the Entrolytics client
type Config struct {
	Endpoint string
	APIKey   string
	Timeout  time.Duration
	Debug    bool
}

// Client represents the Entrolytics Go client
type Client struct {
	config Config
	http   *http.Client
}

// Event represents an analytics event
type Event struct {
	Event       string                 `json:"event"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	UserID      string                 `json:"userId,omitempty"`
	AnonymousID string                 `json:"anonymousId,omitempty"`
	Timestamp   time.Time              `json:"timestamp,omitempty"`
	WebsiteID   string                 `json:"website_id,omitempty"`
}

// New creates a new Entrolytics client
func New(config Config) *Client {
	return &Client{
		config: config,
		http: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Track sends an event to Entrolytics
func (c *Client) Track(event Event) error {
	return c.TrackWithContext(context.Background(), event)
}

// TrackWithContext sends an event to Entrolytics with a context
func (c *Client) TrackWithContext(ctx context.Context, event Event) error {
	// Set default values
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.WebsiteID == "" {
		event.WebsiteID = c.config.APIKey // Assuming APIKey contains website ID
	}

	// Prepare payload
	payload := map[string]interface{}{
		"event":       event.Event,
		"properties":  event.Properties,
		"userId":      event.UserID,
		"anonymousId": event.AnonymousID,
		"timestamp":   event.Timestamp,
		"website_id":  event.WebsiteID,
	}

	// Debug logging
	if c.config.Debug {
		fmt.Printf("Tracking event: %+v\n", payload)
	}

	// Create request
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"/collect", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Send request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return nil
}

// Identify identifies a user
func (c *Client) Identify(userID string, traits map[string]interface{}) error {
	return c.Track(Event{
		Event:      "identify",
		UserID:     userID,
		Properties: traits,
	})
}

// Page tracks a page view
func (c *Client) Page(name string, properties map[string]interface{}) error {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	
	properties["page_name"] = name
	
	return c.Track(Event{
		Event:      "page",
		Properties: properties,
	})
}

// Batch tracks multiple events at once
func (c *Client) Batch(events []Event) error {
	return c.BatchWithContext(context.Background(), events)
}

// BatchWithContext tracks multiple events with context
func (c *Client) BatchWithContext(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	// Prepare batch payload
	batch := make([]map[string]interface{}, len(events))
	for i, event := range events {
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}
		if event.WebsiteID == "" {
			event.WebsiteID = c.config.APIKey
		}

		batch[i] = map[string]interface{}{
			"event":       event.Event,
			"properties":  event.Properties,
			"userId":      event.UserID,
			"anonymousId": event.AnonymousID,
			"timestamp":   event.Timestamp,
			"website_id":  event.WebsiteID,
		}
	}

	// Debug logging
	if c.config.Debug {
		fmt.Printf("Batch tracking %d events\n", len(events))
	}

	// Create request
	jsonPayload, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"/collect/batch", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create batch request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Send request
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error status for batch: %d", resp.StatusCode)
	}

	return nil
}
