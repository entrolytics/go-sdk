package entrolytics

import "fmt"

// Error types for Entrolytics SDK.
var (
	// ErrAPIKeyRequired is returned when the API key is missing.
	ErrAPIKeyRequired = &EntrolyticsError{
		Code:    "api_key_required",
		Message: "API key is required",
	}

	// ErrWebsiteIDRequired is returned when the website ID is missing.
	ErrWebsiteIDRequired = &EntrolyticsError{
		Code:    "website_id_required",
		Message: "website ID is required",
	}

	// ErrEventNameRequired is returned when the event name is missing.
	ErrEventNameRequired = &EntrolyticsError{
		Code:    "event_name_required",
		Message: "event name is required",
	}

	// ErrURLRequired is returned when the URL is missing.
	ErrURLRequired = &EntrolyticsError{
		Code:    "url_required",
		Message: "URL is required",
	}

	// ErrUserIDRequired is returned when the user ID is missing.
	ErrUserIDRequired = &EntrolyticsError{
		Code:    "user_id_required",
		Message: "user ID is required",
	}

	// Phase 2: Web Vitals errors
	// ErrVitalMetricRequired is returned when the vital metric type is missing.
	ErrVitalMetricRequired = &EntrolyticsError{
		Code:    "vital_metric_required",
		Message: "vital metric type is required (LCP, INP, CLS, TTFB, or FCP)",
	}

	// ErrVitalRatingRequired is returned when the vital rating is missing.
	ErrVitalRatingRequired = &EntrolyticsError{
		Code:    "vital_rating_required",
		Message: "vital rating is required (good, needs-improvement, or poor)",
	}

	// Phase 2: Form Analytics errors
	// ErrFormIDRequired is returned when the form ID is missing.
	ErrFormIDRequired = &EntrolyticsError{
		Code:    "form_id_required",
		Message: "form ID is required",
	}

	// ErrFormEventTypeRequired is returned when the form event type is missing.
	ErrFormEventTypeRequired = &EntrolyticsError{
		Code:    "form_event_type_required",
		Message: "form event type is required",
	}

	// ErrURLPathRequired is returned when the URL path is missing.
	ErrURLPathRequired = &EntrolyticsError{
		Code:    "url_path_required",
		Message: "URL path is required",
	}

	// Phase 2: Deployment errors
	// ErrDeployIDRequired is returned when the deployment ID is missing.
	ErrDeployIDRequired = &EntrolyticsError{
		Code:    "deploy_id_required",
		Message: "deployment ID is required",
	}
)

// EntrolyticsError represents an error from the Entrolytics SDK.
type EntrolyticsError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *EntrolyticsError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("entrolytics: %s (status %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("entrolytics: %s", e.Message)
}

// AuthenticationError is returned when the API key is invalid.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	if e.Message == "" {
		return "entrolytics: invalid or missing API key"
	}
	return fmt.Sprintf("entrolytics: %s", e.Message)
}

// RateLimitError is returned when rate limits are exceeded.
type RateLimitError struct {
	Message    string
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("entrolytics: rate limit exceeded, retry after %d seconds", e.RetryAfter)
	}
	return "entrolytics: rate limit exceeded"
}

// NetworkError is returned when a network request fails.
type NetworkError struct {
	Message string
	Err     error
}

func (e *NetworkError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("entrolytics: network error: %v", e.Err)
	}
	return fmt.Sprintf("entrolytics: network error: %s", e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}
