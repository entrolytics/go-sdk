package entrolytics

import (
	"net/http"
	"strings"
)

// PageViewMiddleware creates HTTP middleware that automatically tracks page views.
// It works with any http.Handler compatible router (net/http, chi, etc.).
//
// Example with net/http:
//
//	client := entrolytics.NewClient("ent_xxx")
//	handler := entrolytics.PageViewMiddleware(client, "website_id")(yourHandler)
//	http.ListenAndServe(":8080", handler)
//
// Example with chi:
//
//	r := chi.NewRouter()
//	r.Use(entrolytics.PageViewMiddleware(client, "website_id"))
func PageViewMiddleware(client *Client, websiteID string) func(http.Handler) http.Handler {
	return PageViewMiddlewareWithOptions(client, websiteID, MiddlewareOptions{})
}

// MiddlewareOptions configures the page view middleware.
type MiddlewareOptions struct {
	// SkipPaths are URL paths that should not be tracked.
	SkipPaths []string

	// SkipExtensions are file extensions that should not be tracked.
	// Defaults to common static file extensions if empty.
	SkipExtensions []string

	// TrackQueryParams determines if query parameters are included in the URL.
	TrackQueryParams bool

	// GetUserID is a function to extract user ID from the request.
	GetUserID func(r *http.Request) string

	// GetSessionID is a function to extract session ID from the request.
	GetSessionID func(r *http.Request) string
}

var defaultSkipExtensions = []string{
	".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
	".woff", ".woff2", ".ttf", ".eot", ".map", ".json", ".xml",
}

// PageViewMiddlewareWithOptions creates HTTP middleware with custom options.
func PageViewMiddlewareWithOptions(client *Client, websiteID string, opts MiddlewareOptions) func(http.Handler) http.Handler {
	skipExtensions := opts.SkipExtensions
	if len(skipExtensions) == 0 {
		skipExtensions = defaultSkipExtensions
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only track GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Check skip paths
			for _, path := range opts.SkipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check skip extensions
			for _, ext := range skipExtensions {
				if strings.HasSuffix(r.URL.Path, ext) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Build URL
			url := r.URL.Path
			if opts.TrackQueryParams && r.URL.RawQuery != "" {
				url = url + "?" + r.URL.RawQuery
			}

			// Extract user info
			var userID, sessionID string
			if opts.GetUserID != nil {
				userID = opts.GetUserID(r)
			}
			if opts.GetSessionID != nil {
				sessionID = opts.GetSessionID(r)
			}

			// Track page view (non-blocking)
			go func() {
				_ = client.PageView(PageView{
					WebsiteID: websiteID,
					URL:       url,
					Referrer:  r.Referer(),
					UserAgent: r.UserAgent(),
					IPAddress: getClientIP(r),
					UserID:    userID,
					SessionID: sessionID,
				})
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// TrackEventHandler wraps an http.HandlerFunc to track events.
// Use this for specific endpoints where you want to track custom events.
//
// Example:
//
//	http.HandleFunc("/api/checkout", entrolytics.TrackEventHandler(client, "checkout", nil, checkoutHandler))
func TrackEventHandler(client *Client, websiteID, eventName string, getData func(r *http.Request) map[string]interface{}, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		if getData != nil {
			data = getData(r)
		}

		// Track event (non-blocking)
		go func() {
			_ = client.Track(Event{
				WebsiteID: websiteID,
				Name:      eventName,
				Data:      data,
				URL:       r.URL.Path,
				Referrer:  r.Referer(),
				UserAgent: r.UserAgent(),
				IPAddress: getClientIP(r),
			})
		}()

		handler(w, r)
	}
}

// ResponseRecorder wraps http.ResponseWriter to capture the status code.
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader captures the status code.
func (rr *ResponseRecorder) WriteHeader(code int) {
	rr.StatusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// TrackOnSuccess creates middleware that only tracks page views on successful responses (2xx).
func TrackOnSuccess(client *Client, websiteID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only track GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			rr := &ResponseRecorder{ResponseWriter: w, StatusCode: http.StatusOK}
			next.ServeHTTP(rr, r)

			// Only track successful responses
			if rr.StatusCode >= 200 && rr.StatusCode < 300 {
				go func() {
					_ = client.PageView(PageView{
						WebsiteID: websiteID,
						URL:       r.URL.Path,
						Referrer:  r.Referer(),
						UserAgent: r.UserAgent(),
						IPAddress: getClientIP(r),
					})
				}()
			}
		})
	}
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if there are multiple
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		return cfip
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
