# entro-go

Go SDK for [Entrolytics](https://ng.entrolytics.click) - First-party growth analytics for the edge.

## Installation

```bash
go get github.com/entrolytics/entro-go
```

## Quick Start

```go
package main

import (
    "log"

    entrolytics "github.com/entrolytics/entro-go"
)

func main() {
    // Create client
    client := entrolytics.NewClient("ent_xxx")

    // Track a custom event
    err := client.Track(entrolytics.Event{
        WebsiteID: "abc123",
        Name:      "purchase",
        Data: map[string]interface{}{
            "revenue":  99.99,
            "currency": "USD",
            "product":  "pro-plan",
        },
    })
    if err != nil {
        log.Printf("Failed to track event: %v", err)
    }

    // Track a page view
    err = client.PageView(entrolytics.PageView{
        WebsiteID: "abc123",
        URL:       "/pricing",
        Referrer:  "https://google.com",
        Title:     "Pricing - Entrolytics",
    })
    if err != nil {
        log.Printf("Failed to track page view: %v", err)
    }

    // Identify a user
    err = client.Identify(entrolytics.Identify{
        WebsiteID: "abc123",
        UserID:    "user_456",
        Traits: map[string]interface{}{
            "email":   "user@example.com",
            "plan":    "pro",
            "company": "Acme Inc",
        },
    })
    if err != nil {
        log.Printf("Failed to identify user: %v", err)
    }
}
```

## Collection Endpoints

Entrolytics provides three collection endpoints optimized for different use cases:

### `/api/collect` - Intelligent Routing (Recommended)

The default endpoint that automatically routes to the optimal storage backend based on your plan and website settings.

**Features:**
- Automatic optimization (Free/Pro → Edge, Business/Enterprise → Node.js)
- Zero configuration required
- Best balance of performance and features

**Use when:**
- You want automatic optimization based on your plan
- You're using Entrolytics Cloud
- You don't have specific latency or feature requirements

### `/api/send-native` - Edge Runtime (Fastest)

Direct edge endpoint for sub-50ms global latency.

**Features:**
- Sub-50ms response times globally
- Runs on Vercel Edge Runtime
- Upstash Redis + Neon Serverless
- Best for high-traffic applications

**Limitations:**
- No ClickHouse export
- Basic geo data (country-level)

**Use when:**
- Latency is critical (<50ms required)
- You have high request volume
- You don't need ClickHouse export

### `/api/send` - Node.js Runtime (Full-Featured)

Traditional Node.js endpoint with advanced capabilities.

**Features:**
- ClickHouse export support
- MaxMind GeoIP (city-level accuracy)
- PostgreSQL storage
- Advanced analytics features

**Latency:** 50-150ms (regional)

**Use when:**
- Self-hosted deployments without edge support
- You need ClickHouse data export
- You require city-level geo accuracy
- Custom server-side analytics workflows

## Configuration

### Default (Intelligent Routing)

```go
// Uses /api/collect by default
client := entrolytics.NewClient("ent_xxx")
```

### Edge Runtime Endpoint

```go
// Use edge endpoint for sub-50ms latency
client := entrolytics.NewClientWithOptions(entrolytics.ClientOptions{
    APIKey:   "ent_xxx",
    Host:     "https://ng.entrolytics.click",
    Endpoint: "/api/send-native",
})
```

### Node.js Runtime Endpoint

```go
// Use Node.js endpoint for ClickHouse export and MaxMind GeoIP
client := entrolytics.NewClientWithOptions(entrolytics.ClientOptions{
    APIKey:   "ent_xxx",
    Host:     "https://ng.entrolytics.click",
    Endpoint: "/api/send",
})
```

### Self-Hosted

```go
// For self-hosted Entrolytics instances
client := entrolytics.NewClientWithOptions(entrolytics.ClientOptions{
    APIKey:    "ent_xxx",
    Host:      "https://analytics.yourdomain.com",
    Timeout:   15 * time.Second,
    UserAgent: "my-app/1.0",
})
```

See the [Routing documentation](https://ng.entrolytics.click/docs/concepts/routing) for more details.

## Context Support

All methods support context for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := client.TrackWithContext(ctx, entrolytics.Event{
    WebsiteID: "abc123",
    Name:      "test",
})
```

## HTTP Handler Integration

Track page views from HTTP handlers:

```go
func trackMiddleware(next http.Handler, client *entrolytics.Client, websiteID string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Track page view
        go func() {
            err := client.PageView(entrolytics.PageView{
                WebsiteID: websiteID,
                URL:       r.URL.String(),
                Referrer:  r.Referer(),
                UserAgent: r.UserAgent(),
                IPAddress: getClientIP(r),
            })
            if err != nil {
                log.Printf("Failed to track: %v", err)
            }
        }()

        next.ServeHTTP(w, r)
    })
}

func getClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return strings.Split(xff, ",")[0]
    }
    return strings.Split(r.RemoteAddr, ":")[0]
}
```

## Gin Middleware

```go
func EntrolyticsMiddleware(client *entrolytics.Client, websiteID string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // Track after request completes
        go func() {
            client.PageView(entrolytics.PageView{
                WebsiteID: websiteID,
                URL:       c.Request.URL.String(),
                Referrer:  c.Request.Referer(),
                UserAgent: c.Request.UserAgent(),
                IPAddress: c.ClientIP(),
            })
        }()
    }
}

// Usage
r := gin.Default()
r.Use(EntrolyticsMiddleware(client, "your-website-id"))
```

## Echo Middleware

```go
func EntrolyticsMiddleware(client *entrolytics.Client, websiteID string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            err := next(c)

            // Track after request
            go func() {
                client.PageView(entrolytics.PageView{
                    WebsiteID: websiteID,
                    URL:       c.Request().URL.String(),
                    Referrer:  c.Request().Referer(),
                    UserAgent: c.Request().UserAgent(),
                    IPAddress: c.RealIP(),
                })
            }()

            return err
        }
    }
}
```

## Error Handling

```go
err := client.Track(event)

switch e := err.(type) {
case *entrolytics.AuthenticationError:
    log.Fatal("Invalid API key")

case *entrolytics.RateLimitError:
    log.Printf("Rate limited, retry after %d seconds", e.RetryAfter)

case *entrolytics.NetworkError:
    log.Printf("Network error: %v", e.Unwrap())

case *entrolytics.EntrolyticsError:
    log.Printf("API error: %s (status %d)", e.Message, e.StatusCode)

default:
    if err != nil {
        log.Printf("Unknown error: %v", err)
    }
}
```

## API Reference

### Client Methods

#### `Track(event Event) error`

Track a custom event.

```go
client.Track(entrolytics.Event{
    WebsiteID: "abc123",      // Required
    Name:      "signup",      // Required
    Data:      map[string]interface{}{"plan": "pro"},
    URL:       "/signup",
    Referrer:  "https://google.com",
    UserID:    "user_123",
    UserAgent: "Mozilla/5.0...",
    IPAddress: "192.168.1.1",
})
```

#### `PageView(pv PageView) error`

Track a page view.

```go
client.PageView(entrolytics.PageView{
    WebsiteID: "abc123",      // Required
    URL:       "/pricing",    // Required
    Referrer:  "https://google.com",
    Title:     "Pricing Page",
    UserID:    "user_123",
    UserAgent: "Mozilla/5.0...",
    IPAddress: "192.168.1.1",
})
```

#### `Identify(id Identify) error`

Identify a user with traits.

```go
client.Identify(entrolytics.Identify{
    WebsiteID: "abc123",      // Required
    UserID:    "user_123",    // Required
    Traits: map[string]interface{}{
        "email":   "user@example.com",
        "plan":    "pro",
        "company": "Acme Inc",
    },
})
```

### Types

| Type | Required Fields | Optional Fields |
|------|-----------------|-----------------|
| `Event` | WebsiteID, Name | Data, URL, Referrer, UserID, SessionID, UserAgent, IPAddress, Timestamp |
| `PageView` | WebsiteID, URL | Referrer, Title, UserID, SessionID, UserAgent, IPAddress, Timestamp |
| `Identify` | WebsiteID, UserID | Traits, Timestamp |

### Errors

| Error Type | Description |
|------------|-------------|
| `EntrolyticsError` | Base error type |
| `AuthenticationError` | Invalid API key |
| `RateLimitError` | Rate limit exceeded |
| `NetworkError` | Network request failed |

## License

MIT License - see [LICENSE](LICENSE) for details.
