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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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

	// sessionWindow is the length of a derived session.
	sessionWindow = 30 * time.Minute

	// visitorIdScope values: how long a visitor id identifies the same visitor.
	visitorScopeDaily      = "daily"
	visitorScopePersistent = "persistent"
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
	payload, err := c.eventPayload(&event)
	if err != nil {
		return err
	}

	return c.send(ctx, payload, event.UserAgent, event.IPAddress)
}

// Batch sends up to 100 custom events to Entrolytics in one request.
//
// The API resolves geo data and screens bots once per request, from the forwarded IP address and
// user agent, so every event in a batch must carry the same IPAddress and UserAgent: one visitor's
// events, or server events with neither.
func (c *Client) Batch(events []Event) error {
	return c.BatchWithContext(context.Background(), events)
}

// BatchWithContext sends up to 100 custom events in one request with context for cancellation.
func (c *Client) BatchWithContext(ctx context.Context, events []Event) error {
	if c.apiKey == "" {
		return ErrAPIKeyRequired
	}
	if len(events) == 0 {
		return nil
	}

	visitor := &events[0]
	payloads := make([]collectPayload, 0, len(events))
	for i := range events {
		event := &events[i]
		if event.IPAddress != visitor.IPAddress || event.UserAgent != visitor.UserAgent {
			return ErrBatchVisitorMismatch
		}
		payload, err := c.eventPayload(event)
		if err != nil {
			return err
		}
		payloads = append(payloads, payload)
	}

	return c.sendToEndpoint(ctx, "/collect/batch", batchPayload{Events: payloads}, visitor.UserAgent, visitor.IPAddress)
}

// eventPayload validates a custom event and builds its /collect payload.
func (c *Client) eventPayload(event *Event) (collectPayload, error) {
	if event.WebsiteID == "" {
		return collectPayload{}, ErrWebsiteIDRequired
	}
	if event.Name == "" {
		return collectPayload{}, ErrEventNameRequired
	}

	at := event.Timestamp
	if at.IsZero() {
		at = time.Now()
	}
	sessionID, visitorID, visitorScope := c.visitorIDs(event.WebsiteID, event.SessionID, event.VisitorID, event.IPAddress, event.UserAgent, at)

	properties := map[string]interface{}{}
	for key, value := range event.Data {
		properties[key] = value
	}
	if event.UserID != "" {
		properties["distinctId"] = event.UserID
	}

	return collectPayload{
		WebsiteID:      event.WebsiteID,
		EventID:        generateUUID(),
		Timestamp:      eventTime(at),
		SessionID:      sessionID,
		VisitorID:      visitorID,
		VisitorIDScope: visitorScope,
		URL:            normalizeURL(c.host, event.URL),
		EventType:      "custom_event",
		EventName:      event.Name,
		Referrer:       normalizeReferrer(event.Referrer),
		Properties:     properties,
	}, nil
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

	at := pv.Timestamp
	if at.IsZero() {
		at = time.Now()
	}
	sessionID, visitorID, visitorScope := c.visitorIDs(pv.WebsiteID, pv.SessionID, pv.VisitorID, pv.IPAddress, pv.UserAgent, at)

	properties := map[string]interface{}{}
	if pv.Title != "" {
		properties["title"] = pv.Title
	}
	if pv.UserID != "" {
		properties["distinctId"] = pv.UserID
	}

	payload := collectPayload{
		WebsiteID:      pv.WebsiteID,
		EventID:        generateUUID(),
		Timestamp:      eventTime(at),
		SessionID:      sessionID,
		VisitorID:      visitorID,
		VisitorIDScope: visitorScope,
		URL:            normalizeURL(c.host, pv.URL),
		EventType:      "pageview",
		Referrer:       normalizeReferrer(pv.Referrer),
		Properties:     properties,
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

	visitorID, visitorScope := id.VisitorID, visitorScopePersistent
	if visitorID == "" {
		visitorID, visitorScope = generateUUID(), ""
	}

	properties := map[string]interface{}{}
	for key, value := range id.Traits {
		properties[key] = value
	}
	properties["distinctId"] = id.UserID
	properties["identify"] = true

	payload := collectPayload{
		WebsiteID:      id.WebsiteID,
		EventID:        generateUUID(),
		Timestamp:      eventTime(id.Timestamp),
		SessionID:      sessionID,
		VisitorID:      visitorID,
		VisitorIDScope: visitorScope,
		URL:            normalizeURL(c.host, "/identify"),
		EventType:      "custom_event",
		EventName:      "identify",
		Properties:     properties,
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

// visitorIDs returns the session and visitor ids for an event at the given time: the caller's,
// else ones derived from the visitor's IP address and user agent, else new random ones when
// neither is known.
//
// It also returns the visitor id's visitorIdScope, how long it identifies the visitor; the API's
// retention report counts only persistent ids. The caller's id is persistent, a derived id rotates
// daily, and a random id has no defined lifetime, so its scope is empty and left out.
func (c *Client) visitorIDs(websiteID, sessionID, visitorID, ipAddress, userAgent string, at time.Time) (session, visitor, visitorScope string) {
	known := ipAddress != "" || userAgent != ""
	session, visitor = sessionID, visitorID
	if session == "" {
		if known {
			session = deriveSessionID(c.apiKey, websiteID, ipAddress, userAgent, at)
		} else {
			session = generateUUID()
		}
	}
	switch {
	case visitor != "":
		visitorScope = visitorScopePersistent
	case known:
		visitor = deriveVisitorID(c.apiKey, websiteID, ipAddress, userAgent, at)
		visitorScope = visitorScopeDaily
	default:
		visitor = generateUUID()
	}
	return session, visitor, visitorScope
}

// deriveVisitorID returns a site-scoped visitor id that rotates at midnight UTC, derived from the
// visitor's IP address and user agent. A new id per call counted every server-side event as a new
// visitor. The API identifies browser visitors the same way, with a daily keyed hash and no
// cookie; the key is the API key, so the id cannot be recomputed from an IP address and user
// agent alone.
func deriveVisitorID(apiKey, websiteID, ipAddress, userAgent string, at time.Time) string {
	day := at.UTC().Format("2006-01-02")
	return hmacUUID(apiKey, websiteID+"\x00"+day+"\x00"+ipAddress+"\x00"+userAgent)
}

// deriveSessionID returns a session id for the visitor's 30-minute UTC window (starting on the
// hour and the half hour), derived like the visitor id from the IP address and user agent, with
// no cookie. A new id per call counted every server-side event as a new visit. A visit that
// crosses a window boundary counts as two sessions.
func deriveSessionID(apiKey, websiteID, ipAddress, userAgent string, at time.Time) string {
	slot := at.UTC().Truncate(sessionWindow).Format("2006-01-02T15:04")
	return hmacUUID(apiKey, websiteID+"\x00session\x00"+slot+"\x00"+ipAddress+"\x00"+userAgent)
}

// hmacUUID formats the first 16 bytes of HMAC-SHA256(key, message) as a version 4 UUID.
func hmacUUID(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return formatUUIDv4(mac.Sum(nil)[:16])
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012x", time.Now().UnixNano()&0xFFFFFFFFFFFF)
	}

	return formatUUIDv4(b)
}

// formatUUIDv4 sets the version 4 and RFC 4122 variant bits on 16 bytes and formats them.
func formatUUIDv4(b []byte) string {
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
