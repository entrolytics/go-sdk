// Package entrolytics provides a Go SDK for Entrolytics analytics.
//
// Entrolytics is a first-party growth analytics platform for the edge.
// This SDK enables server-side event tracking from Go applications.
//
// Basic usage:
//
//	client := entrolytics.NewClient("ent_xxx")
//
//	// Track an event
//	err := client.Track(entrolytics.Event{
//	    WebsiteID: "abc123",
//	    Name:      "purchase",
//	    Data: map[string]interface{}{
//	        "revenue":  99.99,
//	        "currency": "USD",
//	    },
//	})
//
//	// Track a page view
//	err = client.PageView(entrolytics.PageView{
//	    WebsiteID: "abc123",
//	    URL:       "/pricing",
//	})
//
//	// Identify a user
//	err = client.Identify(entrolytics.Identify{
//	    WebsiteID: "abc123",
//	    UserID:    "user_456",
//	    Traits: map[string]interface{}{
//	        "email": "user@example.com",
//	        "plan":  "pro",
//	    },
//	})
package entrolytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	// DefaultHost is the default Entrolytics API host.
	DefaultHost = "https://entrolytics.click"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 10 * time.Second

	// Version is the SDK version.
	Version = "2.1.0"
)

// Client is the Entrolytics API client.
type Client struct {
	apiKey    string
	host      string
	timeout   time.Duration
	userAgent string
	http      *http.Client
}

// NewClient creates a new Entrolytics client with the given API key.
func NewClient(apiKey string) *Client {
	return NewClientWithOptions(ClientOptions{
		APIKey: apiKey,
	})
}

// NewClientWithOptions creates a new Entrolytics client with custom options.
func NewClientWithOptions(opts ClientOptions) *Client {
	if opts.Host == "" {
		opts.Host = DefaultHost
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.UserAgent == "" {
		opts.UserAgent = fmt.Sprintf("entrolytics-go/%s", Version)
	}

	return &Client{
		apiKey:    opts.APIKey,
		host:      opts.Host,
		timeout:   opts.Timeout,
		userAgent: opts.UserAgent,
		http: &http.Client{
			Timeout: opts.Timeout,
		},
	}
}

// Track sends a custom event to Entrolytics.
func (c *Client) Track(event Event) error {
	return c.TrackWithContext(context.Background(), event)
}

// TrackWithContext sends a custom event with context for cancellation.
func (c *Client) TrackWithContext(ctx context.Context, event Event) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if event.WebsiteID == "" {
		return ErrWebsiteIDRequired
	}
	if event.Name == "" {
		return ErrEventNameRequired
	}

	timestamp := event.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	payload := eventPayload{
		Type: "event",
		Payload: trackPayload{
			Website:   event.WebsiteID,
			Name:      event.Name,
			Data:      event.Data,
			URL:       event.URL,
			Referrer:  event.Referrer,
			UserID:    event.UserID,
			SessionID: event.SessionID,
			Timestamp: timestamp.Format(time.RFC3339),
		},
	}

	return c.send(ctx, payload, event.UserAgent, event.IPAddress)
}

// PageView sends a page view event to Entrolytics.
func (c *Client) PageView(pv PageView) error {
	return c.PageViewWithContext(context.Background(), pv)
}

// PageViewWithContext sends a page view with context for cancellation.
func (c *Client) PageViewWithContext(ctx context.Context, pv PageView) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if pv.WebsiteID == "" {
		return ErrWebsiteIDRequired
	}
	if pv.URL == "" {
		return ErrURLRequired
	}

	timestamp := pv.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	data := make(map[string]interface{})
	if pv.Title != "" {
		data["title"] = pv.Title
	}

	payload := eventPayload{
		Type: "event",
		Payload: trackPayload{
			Website:   pv.WebsiteID,
			Name:      "$pageview",
			Data:      data,
			URL:       pv.URL,
			Referrer:  pv.Referrer,
			UserID:    pv.UserID,
			SessionID: pv.SessionID,
			Timestamp: timestamp.Format(time.RFC3339),
		},
	}

	return c.send(ctx, payload, pv.UserAgent, pv.IPAddress)
}

// Identify sends user identification data to Entrolytics.
func (c *Client) Identify(id Identify) error {
	return c.IdentifyWithContext(context.Background(), id)
}

// IdentifyWithContext sends user identification with context for cancellation.
func (c *Client) IdentifyWithContext(ctx context.Context, id Identify) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if id.WebsiteID == "" {
		return ErrWebsiteIDRequired
	}
	if id.UserID == "" {
		return ErrUserIDRequired
	}

	timestamp := id.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	payload := eventPayload{
		Type: "identify",
		Payload: identifyPayload{
			Website:   id.WebsiteID,
			UserID:    id.UserID,
			Traits:    id.Traits,
			Timestamp: timestamp.Format(time.RFC3339),
		},
	}

	return c.send(ctx, payload, "", "")
}

// ============================================================================
// Phase 2: Web Vitals (requires entrolytics)
// ============================================================================

// TrackVital sends a Web Vital metric to Entrolytics.
// Note: This feature requires entrolytics.
func (c *Client) TrackVital(vital WebVital) error {
	return c.TrackVitalWithContext(context.Background(), vital)
}

// TrackVitalWithContext sends a Web Vital metric with context for cancellation.
func (c *Client) TrackVitalWithContext(ctx context.Context, vital WebVital) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if vital.WebsiteID == "" {
		return ErrWebsiteIDRequired
	}
	if vital.Metric == "" {
		return ErrVitalMetricRequired
	}
	if vital.Rating == "" {
		return ErrVitalRatingRequired
	}

	payload := vitalPayload{
		Website:        vital.WebsiteID,
		Metric:         vital.Metric,
		Value:          vital.Value,
		Rating:         vital.Rating,
		Delta:          vital.Delta,
		ID:             vital.ID,
		NavigationType: vital.NavigationType,
		Attribution:    vital.Attribution,
		URL:            vital.URL,
		Path:           vital.Path,
		SessionID:      vital.SessionID,
	}

	return c.sendToEndpoint(ctx, "/api/collect/vitals", payload, "", "")
}

// ============================================================================
// Phase 2: Form Analytics (requires entrolytics)
// ============================================================================

// TrackFormEvent sends a form interaction event to Entrolytics.
// Note: This feature requires entrolytics.
func (c *Client) TrackFormEvent(event FormEvent) error {
	return c.TrackFormEventWithContext(context.Background(), event)
}

// TrackFormEventWithContext sends a form event with context for cancellation.
func (c *Client) TrackFormEventWithContext(ctx context.Context, event FormEvent) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if event.WebsiteID == "" {
		return ErrWebsiteIDRequired
	}
	if event.FormID == "" {
		return ErrFormIDRequired
	}
	if event.EventType == "" {
		return ErrFormEventTypeRequired
	}
	if event.URLPath == "" {
		return ErrURLPathRequired
	}

	payload := formEventPayload{
		Website:        event.WebsiteID,
		EventType:      event.EventType,
		FormID:         event.FormID,
		FormName:       event.FormName,
		URLPath:        event.URLPath,
		FieldName:      event.FieldName,
		FieldType:      event.FieldType,
		FieldIndex:     event.FieldIndex,
		TimeOnField:    event.TimeOnField,
		TimeSinceStart: event.TimeSinceStart,
		ErrorMessage:   event.ErrorMessage,
		Success:        event.Success,
		SessionID:      event.SessionID,
	}

	return c.sendToEndpoint(ctx, "/api/collect/forms", payload, "", "")
}

// ============================================================================
// Phase 2: Deployment Tracking (requires entrolytics)
// ============================================================================

// SetDeployment registers deployment context with Entrolytics.
// Note: This feature requires entrolytics.
func (c *Client) SetDeployment(deploy Deployment) error {
	return c.SetDeploymentWithContext(context.Background(), deploy)
}

// SetDeploymentWithContext registers deployment with context for cancellation.
func (c *Client) SetDeploymentWithContext(ctx context.Context, deploy Deployment) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if deploy.WebsiteID == "" {
		return ErrWebsiteIDRequired
	}
	if deploy.DeployID == "" {
		return ErrDeployIDRequired
	}

	payload := deploymentPayload{
		Website:   deploy.WebsiteID,
		DeployID:  deploy.DeployID,
		GitSha:    deploy.GitSha,
		GitBranch: deploy.GitBranch,
		DeployURL: deploy.DeployURL,
		Source:    deploy.Source,
	}

	return c.sendToEndpoint(ctx, fmt.Sprintf("/api/websites/%s/deployments", deploy.WebsiteID), payload, "", "")
}

// send performs the HTTP request to the Entrolytics API.
func (c *Client) send(ctx context.Context, payload interface{}, userAgent, ipAddress string) error {
	return c.sendToEndpoint(ctx, "/api/send", payload, userAgent, ipAddress)
}

// sendToEndpoint performs the HTTP request to a specific endpoint.
func (c *Client) sendToEndpoint(ctx context.Context, endpoint string, payload interface{}, userAgent, ipAddress string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &NetworkError{Message: "failed to marshal payload", Err: err}
	}

	url := fmt.Sprintf("%s%s", c.host, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &NetworkError{Message: "failed to create request", Err: err}
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if userAgent != "" {
		req.Header.Set("X-Forwarded-User-Agent", userAgent)
	}
	if ipAddress != "" {
		req.Header.Set("X-Forwarded-For", ipAddress)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &NetworkError{Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	return c.handleResponse(resp)
}

// handleResponse processes the API response.
func (c *Client) handleResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &AuthenticationError{}

	case http.StatusBadRequest:
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return &EntrolyticsError{
				Code:       "validation_error",
				Message:    errResp.Error,
				StatusCode: resp.StatusCode,
			}
		}
		return &EntrolyticsError{
			Code:       "bad_request",
			Message:    "invalid request",
			StatusCode: resp.StatusCode,
		}

	case http.StatusTooManyRequests:
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			var err error
			retryAfter, err = strconv.Atoi(ra)
			if err != nil {
				retryAfter = 0
			}
		}
		return &RateLimitError{RetryAfter: retryAfter}

	default:
		return &EntrolyticsError{
			Code:       "request_failed",
			Message:    fmt.Sprintf("request failed with status %d", resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
	}
}
