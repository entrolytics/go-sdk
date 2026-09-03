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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultHost is the default Entrolytics API host.
	DefaultHost = "https://api.entrolytics.click"

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

	sessionID := event.SessionID
	if sessionID == "" {
		sessionID = generateUUID()
	}

	visitorID := event.VisitorID
	if visitorID == "" {
		visitorID = generateUUID()
	}

	properties := map[string]interface{}{}
	for key, value := range event.Data {
		properties[key] = value
	}
	if event.UserID != "" {
		properties["distinctId"] = event.UserID
	}

	payload := collectPayload{
		WebsiteID:  event.WebsiteID,
		EventID:    generateUUID(),
		Timestamp:  eventTime(event.Timestamp),
		SessionID:  sessionID,
		VisitorID:  visitorID,
		URL:        normalizeURL(c.host, event.URL),
		EventType:  "custom_event",
		EventName:  event.Name,
		Referrer:   normalizeReferrer(event.Referrer),
		Properties: properties,
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

	sessionID := pv.SessionID
	if sessionID == "" {
		sessionID = generateUUID()
	}

	visitorID := pv.VisitorID
	if visitorID == "" {
		visitorID = generateUUID()
	}

	properties := map[string]interface{}{}
	if pv.Title != "" {
		properties["title"] = pv.Title
	}
	if pv.UserID != "" {
		properties["distinctId"] = pv.UserID
	}

	payload := collectPayload{
		WebsiteID:  pv.WebsiteID,
		EventID:    generateUUID(),
		Timestamp:  eventTime(pv.Timestamp),
		SessionID:  sessionID,
		VisitorID:  visitorID,
		URL:        normalizeURL(c.host, pv.URL),
		EventType:  "pageview",
		Referrer:   normalizeReferrer(pv.Referrer),
		Properties: properties,
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

	sessionID := id.SessionID
	if sessionID == "" {
		sessionID = generateUUID()
	}

	visitorID := id.VisitorID
	if visitorID == "" {
		visitorID = generateUUID()
	}

	properties := map[string]interface{}{}
	for key, value := range id.Traits {
		properties[key] = value
	}
	properties["distinctId"] = id.UserID
	properties["identify"] = true

	payload := collectPayload{
		WebsiteID:  id.WebsiteID,
		EventID:    generateUUID(),
		Timestamp:  eventTime(id.Timestamp),
		SessionID:  sessionID,
		VisitorID:  visitorID,
		URL:        normalizeURL(c.host, "/identify"),
		EventType:  "custom_event",
		EventName:  "identify",
		Properties: properties,
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
		WebsiteID:   vital.WebsiteID,
		VisitorID:   firstNonEmpty(vital.VisitorID, generateUUID()),
		SessionID:   firstNonEmpty(vital.SessionID, generateUUID()),
		URL:         normalizeURL(c.host, firstNonEmpty(vital.URL, "/vital")),
		Path:        firstNonEmpty(vital.Path, "/vital"),
		MetricName:  vital.Metric,
		MetricValue: vital.Value,
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
		WebsiteID:      event.WebsiteID,
		VisitorID:      firstNonEmpty(event.VisitorID, generateUUID()),
		SessionID:      firstNonEmpty(event.SessionID, generateUUID()),
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

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func eventTime(value time.Time) string {
	if value.IsZero() {
		value = time.Now()
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012x", time.Now().UnixNano()&0xFFFFFFFFFFFF)
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%04x%08x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:12],
		b[12:16],
	)
}

func normalizeURL(host, rawURL string) string {
	if rawURL == "" {
		rawURL = "/event"
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}

	if !strings.HasPrefix(rawURL, "/") {
		rawURL = "/" + rawURL
	}

	return strings.TrimRight(host, "/") + rawURL
}

func normalizeReferrer(rawReferrer string) string {
	if strings.HasPrefix(rawReferrer, "http://") || strings.HasPrefix(rawReferrer, "https://") {
		return rawReferrer
	}

	return ""
}

// send performs the HTTP request to the Entrolytics API.
func (c *Client) send(ctx context.Context, payload interface{}, userAgent, ipAddress string) error {
	return c.sendToEndpoint(ctx, "/collect", payload, userAgent, ipAddress)
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

	req.Header.Set("x-api-key", c.apiKey)
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
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
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
