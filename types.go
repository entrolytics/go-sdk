package entrolytics

import "time"

// Event represents a custom tracking event.
type Event struct {
	// WebsiteID is your Entrolytics website ID (required).
	WebsiteID string

	// Name is the event name, e.g., "purchase", "signup" (required).
	Name string

	// Data contains additional event data.
	Data map[string]interface{}

	// URL is the page URL where the event occurred.
	URL string

	// Referrer is the referrer URL.
	Referrer string

	// UserID identifies a logged-in user.
	UserID string

	// SessionID identifies the user session.
	SessionID string

	// UserAgent is the client's user agent string.
	UserAgent string

	// IPAddress is the client's IP address for geo data.
	IPAddress string

	// Timestamp is when the event occurred. Defaults to now if empty.
	Timestamp time.Time
}

// PageView represents a page view event.
type PageView struct {
	// WebsiteID is your Entrolytics website ID (required).
	WebsiteID string

	// URL is the page URL (required).
	URL string

	// Referrer is the referrer URL.
	Referrer string

	// Title is the page title.
	Title string

	// UserID identifies a logged-in user.
	UserID string

	// SessionID identifies the user session.
	SessionID string

	// UserAgent is the client's user agent string.
	UserAgent string

	// IPAddress is the client's IP address.
	IPAddress string

	// Timestamp is when the page view occurred.
	Timestamp time.Time
}

// Identify represents user identification data.
type Identify struct {
	// WebsiteID is your Entrolytics website ID (required).
	WebsiteID string

	// UserID is the unique user identifier (required).
	UserID string

	// Traits are user attributes like email, plan, company.
	Traits map[string]interface{}

	// Timestamp is when the identification occurred.
	Timestamp time.Time
}

// Response represents the API response.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ClientOptions configures the Entrolytics client.
type ClientOptions struct {
	// APIKey is your Entrolytics API key (required).
	APIKey string

	// Host is the Entrolytics API host. Defaults to https://entrolytics.click.
	Host string

	// Timeout is the HTTP request timeout. Defaults to 10 seconds.
	Timeout time.Duration

	// UserAgent is the User-Agent header for requests.
	UserAgent string
}

// eventPayload is the internal structure for sending events.
type eventPayload struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type trackPayload struct {
	Website   string                 `json:"website"`
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data,omitempty"`
	URL       string                 `json:"url,omitempty"`
	Referrer  string                 `json:"referrer,omitempty"`
	UserID    string                 `json:"userId,omitempty"`
	SessionID string                 `json:"sessionId,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

type identifyPayload struct {
	Website   string                 `json:"website"`
	UserID    string                 `json:"userId"`
	Traits    map[string]interface{} `json:"traits,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// ============================================================================
// Phase 2: Web Vitals Types
// ============================================================================

// VitalMetric represents a Core Web Vital metric type.
type VitalMetric string

const (
	// LCP is Largest Contentful Paint (Core Web Vital).
	LCP VitalMetric = "LCP"
	// INP is Interaction to Next Paint (Core Web Vital, replaced FID).
	INP VitalMetric = "INP"
	// CLS is Cumulative Layout Shift (Core Web Vital).
	CLS VitalMetric = "CLS"
	// TTFB is Time to First Byte.
	TTFB VitalMetric = "TTFB"
	// FCP is First Contentful Paint.
	FCP VitalMetric = "FCP"
)

// VitalRating represents a Web Vital performance rating.
type VitalRating string

const (
	// Good indicates the metric meets performance standards.
	Good VitalRating = "good"
	// NeedsImprovement indicates the metric needs optimization.
	NeedsImprovement VitalRating = "needs-improvement"
	// Poor indicates poor performance.
	Poor VitalRating = "poor"
)

// NavigationType represents how the page was navigated to.
type NavigationType string

const (
	Navigate         NavigationType = "navigate"
	Reload           NavigationType = "reload"
	BackForward      NavigationType = "back-forward"
	BackForwardCache NavigationType = "back-forward-cache"
	Prerender        NavigationType = "prerender"
	Restore          NavigationType = "restore"
)

// WebVital represents a Web Vital metric to track.
// Note: Web Vitals tracking requires entrolytics.
type WebVital struct {
	// WebsiteID is your Entrolytics website ID (required).
	WebsiteID string

	// Metric is the vital type: LCP, INP, CLS, TTFB, or FCP (required).
	Metric VitalMetric

	// Value is the metric value in milliseconds (except CLS which is unitless) (required).
	Value float64

	// Rating indicates performance: good, needs-improvement, or poor (required).
	Rating VitalRating

	// Delta is the difference from the previous value.
	Delta float64

	// ID is a unique identifier for deduplication.
	ID string

	// NavigationType indicates how the page was navigated to.
	NavigationType NavigationType

	// Attribution provides debug information about the metric.
	Attribution map[string]interface{}

	// URL is the full page URL.
	URL string

	// Path is the URL path component.
	Path string

	// SessionID identifies the user session.
	SessionID string

	// Timestamp is when the metric was recorded.
	Timestamp time.Time
}

type vitalPayload struct {
	Website        string                 `json:"website"`
	Metric         VitalMetric            `json:"metric"`
	Value          float64                `json:"value"`
	Rating         VitalRating            `json:"rating"`
	Delta          float64                `json:"delta,omitempty"`
	ID             string                 `json:"id,omitempty"`
	NavigationType NavigationType         `json:"navigationType,omitempty"`
	Attribution    map[string]interface{} `json:"attribution,omitempty"`
	URL            string                 `json:"url,omitempty"`
	Path           string                 `json:"path,omitempty"`
	SessionID      string                 `json:"sessionId,omitempty"`
}

// ============================================================================
// Phase 2: Form Analytics Types
// ============================================================================

// FormEventType represents a form interaction event type.
type FormEventType string

const (
	// FormStart indicates form interaction started.
	FormStart FormEventType = "start"
	// FieldFocus indicates a field received focus.
	FieldFocus FormEventType = "field_focus"
	// FieldBlur indicates a field lost focus.
	FieldBlur FormEventType = "field_blur"
	// FieldError indicates a field validation error.
	FieldError FormEventType = "field_error"
	// FormSubmit indicates form was submitted.
	FormSubmit FormEventType = "submit"
	// FormAbandon indicates form was abandoned.
	FormAbandon FormEventType = "abandon"
)

// FormEvent represents a form interaction event to track.
// Note: Form tracking requires entrolytics.
type FormEvent struct {
	// WebsiteID is your Entrolytics website ID (required).
	WebsiteID string

	// EventType is the form event type (required).
	EventType FormEventType

	// FormID is a unique identifier for the form (required).
	FormID string

	// FormName is the human-readable form name.
	FormName string

	// URLPath is the page path where the form is located (required).
	URLPath string

	// FieldName is the name of the field (for field events).
	FieldName string

	// FieldType is the input type (text, email, select, etc.).
	FieldType string

	// FieldIndex is the position of the field in the form.
	FieldIndex int

	// TimeOnField is milliseconds spent on the field.
	TimeOnField int

	// TimeSinceStart is milliseconds since form interaction started.
	TimeSinceStart int

	// ErrorMessage is the validation error message (for error events).
	ErrorMessage string

	// Success indicates whether the submission was successful.
	Success bool

	// SessionID identifies the user session.
	SessionID string

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

type formEventPayload struct {
	Website        string        `json:"website"`
	EventType      FormEventType `json:"eventType"`
	FormID         string        `json:"formId"`
	FormName       string        `json:"formName,omitempty"`
	URLPath        string        `json:"urlPath"`
	FieldName      string        `json:"fieldName,omitempty"`
	FieldType      string        `json:"fieldType,omitempty"`
	FieldIndex     int           `json:"fieldIndex,omitempty"`
	TimeOnField    int           `json:"timeOnField,omitempty"`
	TimeSinceStart int           `json:"timeSinceStart,omitempty"`
	ErrorMessage   string        `json:"errorMessage,omitempty"`
	Success        bool          `json:"success,omitempty"`
	SessionID      string        `json:"sessionId,omitempty"`
}

// ============================================================================
// Phase 2: Deployment Types
// ============================================================================

// DeploymentSource represents the deployment platform.
type DeploymentSource string

const (
	Vercel     DeploymentSource = "vercel"
	Netlify    DeploymentSource = "netlify"
	Cloudflare DeploymentSource = "cloudflare"
	Railway    DeploymentSource = "railway"
	Render     DeploymentSource = "render"
	Fly        DeploymentSource = "fly"
	Heroku     DeploymentSource = "heroku"
	AWS        DeploymentSource = "aws"
	GCP        DeploymentSource = "gcp"
	Azure      DeploymentSource = "azure"
	Custom     DeploymentSource = "custom"
)

// Deployment represents deployment context to register.
// Note: Deployment tracking requires entrolytics.
type Deployment struct {
	// WebsiteID is your Entrolytics website ID (required).
	WebsiteID string

	// DeployID is a unique identifier for this deployment (required).
	DeployID string

	// GitSha is the git commit SHA.
	GitSha string

	// GitBranch is the git branch name.
	GitBranch string

	// DeployURL is the deployment URL.
	DeployURL string

	// Source is the deployment platform.
	Source DeploymentSource
}

type deploymentPayload struct {
	Website   string           `json:"website"`
	DeployID  string           `json:"deployId"`
	GitSha    string           `json:"gitSha,omitempty"`
	GitBranch string           `json:"gitBranch,omitempty"`
	DeployURL string           `json:"deployUrl,omitempty"`
	Source    DeploymentSource `json:"source,omitempty"`
}
