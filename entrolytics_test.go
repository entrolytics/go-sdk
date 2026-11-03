package entrolytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testAPIKey    = "ent_test"
	testWebsiteID = "6f1c7c8e-5d3b-4f5a-9d1e-2b6f0c9a7e41"
	testIP        = "203.0.113.7"
	testUserAgent = "Mozilla/5.0 (Macintosh) Safari/605.1.15"

	visitorVector      = "cb36ddfc-3c30-41b4-8d6a-f3928d890509"
	firstSessionVector = "6e75928f-813d-49ee-a4dc-5d2e299d7eb2"
	nextSessionVector  = "f656564a-ee6c-4cc1-8309-0b69c73bfd3d"
)

func at(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// collect starts an API stub and returns a client for it and the payloads it received.
func collect(t *testing.T) (*Client, *[]collectPayload) {
	t.Helper()
	var payloads []collectPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload collectPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		payloads = append(payloads, payload)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	return NewClientWithOptions(ClientOptions{APIKey: testAPIKey, Host: server.URL}), &payloads
}

func TestDerivedIDsMatchTheOtherSDKs(t *testing.T) {
	if got := deriveVisitorID(testAPIKey, testWebsiteID, testIP, testUserAgent, at(t, "2026-09-24T14:30:00Z")); got != visitorVector {
		t.Errorf("visitor id = %s, want %s", got, visitorVector)
	}
	for _, value := range []string{"2026-09-24T14:30:00Z", "2026-09-24T14:47:00Z", "2026-09-24T14:59:59Z"} {
		if got := deriveSessionID(testAPIKey, testWebsiteID, testIP, testUserAgent, at(t, value)); got != firstSessionVector {
			t.Errorf("session id at %s = %s, want %s", value, got, firstSessionVector)
		}
	}
	if got := deriveSessionID(testAPIKey, testWebsiteID, testIP, testUserAgent, at(t, "2026-09-24T15:00:00Z")); got != nextSessionVector {
		t.Errorf("session id at 15:00 = %s, want %s", got, nextSessionVector)
	}
}

func TestEventsOfOneVisitorShareTheSession(t *testing.T) {
	client, payloads := collect(t)

	for _, value := range []string{"2026-09-24T14:30:00Z", "2026-09-24T14:59:59Z", "2026-09-24T15:00:00Z"} {
		if err := client.PageView(PageView{WebsiteID: testWebsiteID, URL: "/pricing", IPAddress: testIP, UserAgent: testUserAgent, Timestamp: at(t, value)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.Track(Event{WebsiteID: testWebsiteID, Name: "signup", IPAddress: testIP, UserAgent: testUserAgent, Timestamp: at(t, "2026-09-24T14:47:00Z")}); err != nil {
		t.Fatal(err)
	}

	want := []string{firstSessionVector, firstSessionVector, nextSessionVector, firstSessionVector}
	if len(*payloads) != len(want) {
		t.Fatalf("received %d events, want %d", len(*payloads), len(want))
	}
	for i, payload := range *payloads {
		if payload.SessionID != want[i] {
			t.Errorf("event %d session id = %s, want %s", i, payload.SessionID, want[i])
		}
		if payload.VisitorID != visitorVector {
			t.Errorf("event %d visitor id = %s, want %s", i, payload.VisitorID, visitorVector)
		}
	}
}

func TestExplicitIDsWinAndUnknownVisitorsGetNewIDs(t *testing.T) {
	client, payloads := collect(t)
	when := at(t, "2026-09-24T14:30:00Z")

	if err := client.Track(Event{WebsiteID: testWebsiteID, Name: "signup", SessionID: "session", VisitorID: "visitor", IPAddress: testIP, UserAgent: testUserAgent, Timestamp: when}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := client.Track(Event{WebsiteID: testWebsiteID, Name: "signup", Timestamp: when}); err != nil {
			t.Fatal(err)
		}
	}

	if len(*payloads) != 3 {
		t.Fatalf("received %d events, want 3", len(*payloads))
	}
	explicit, first, second := (*payloads)[0], (*payloads)[1], (*payloads)[2]
	if explicit.SessionID != "session" || explicit.VisitorID != "visitor" {
		t.Errorf("explicit ids = %s/%s, want session/visitor", explicit.SessionID, explicit.VisitorID)
	}
	if first.SessionID == second.SessionID || first.VisitorID == second.VisitorID {
		t.Errorf("events without visitor context shared ids: %+v %+v", first, second)
	}
}

func TestBatchSendsTheEventsOfOneVisitorInOneRequest(t *testing.T) {
	var requests []*http.Request
	var bodies []batchPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body batchPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode batch: %v", err)
		}
		requests = append(requests, r)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	client := NewClientWithOptions(ClientOptions{APIKey: testAPIKey, Host: server.URL})
	when := at(t, "2026-09-24T14:47:00Z")
	event := Event{WebsiteID: testWebsiteID, Name: "signup", IPAddress: testIP, UserAgent: testUserAgent, Timestamp: when}

	if err := client.Batch(nil); err != nil {
		t.Fatal(err)
	}
	if err := client.Batch([]Event{event, {WebsiteID: testWebsiteID, Name: "upgrade", IPAddress: testIP, UserAgent: testUserAgent, Timestamp: when}}); err != nil {
		t.Fatal(err)
	}

	if len(requests) != 1 {
		t.Fatalf("sent %d requests, want 1", len(requests))
	}
	request, events := requests[0], bodies[0].Events
	if request.URL.Path != "/collect/batch" || request.Header.Get("x-api-key") != testAPIKey {
		t.Errorf("request = %s with key %q", request.URL.Path, request.Header.Get("x-api-key"))
	}
	if request.Header.Get("X-Forwarded-For") != testIP || request.Header.Get("X-Forwarded-User-Agent") != testUserAgent {
		t.Errorf("forwarded visitor = %q/%q", request.Header.Get("X-Forwarded-For"), request.Header.Get("X-Forwarded-User-Agent"))
	}
	if len(events) != 2 || events[0].EventName != "signup" || events[1].EventName != "upgrade" {
		t.Fatalf("events = %+v", events)
	}
	if events[0].EventID == events[1].EventID {
		t.Errorf("events share the event id %s", events[0].EventID)
	}
	for i, payload := range events {
		if payload.SessionID != firstSessionVector || payload.VisitorID != visitorVector || payload.VisitorIDScope != "daily" || payload.EventType != "custom_event" {
			t.Errorf("event %d = %+v", i, payload)
		}
	}
}

func TestBatchRejectsInvalidEventsAndMixedVisitors(t *testing.T) {
	client, payloads := collect(t)
	event := Event{WebsiteID: testWebsiteID, Name: "signup", IPAddress: testIP, UserAgent: testUserAgent}

	if err := client.Batch([]Event{event, {WebsiteID: testWebsiteID, Name: "signup", IPAddress: "198.51.100.4", UserAgent: testUserAgent}}); err != ErrBatchVisitorMismatch {
		t.Errorf("mixed visitors: err = %v, want ErrBatchVisitorMismatch", err)
	}
	if err := client.Batch([]Event{event, {WebsiteID: testWebsiteID, IPAddress: testIP, UserAgent: testUserAgent}}); err != ErrEventNameRequired {
		t.Errorf("unnamed event: err = %v, want ErrEventNameRequired", err)
	}
	if len(*payloads) != 0 {
		t.Errorf("sent %d requests for invalid batches, want 0", len(*payloads))
	}
}

func TestEventsDeclareHowLongTheVisitorIDLasts(t *testing.T) {
	client, payloads := collect(t)
	derived := Event{WebsiteID: testWebsiteID, Name: "signup", IPAddress: testIP, UserAgent: testUserAgent}

	steps := []func() error{
		func() error { return client.Track(derived) },
		func() error {
			return client.PageView(PageView{WebsiteID: testWebsiteID, URL: "/pricing", UserAgent: testUserAgent})
		},
		func() error {
			return client.Track(Event{WebsiteID: testWebsiteID, Name: "signup", VisitorID: "customer-42", IPAddress: testIP})
		},
		func() error {
			return client.Identify(Identify{WebsiteID: testWebsiteID, UserID: "user-1", VisitorID: "customer-42"})
		},
		func() error { return client.Track(Event{WebsiteID: testWebsiteID, Name: "signup"}) },
		func() error { return client.Identify(Identify{WebsiteID: testWebsiteID, UserID: "user-1"}) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{"daily", "daily", "persistent", "persistent", "", ""}
	if len(*payloads) != len(want) {
		t.Fatalf("received %d events, want %d", len(*payloads), len(want))
	}
	for i, payload := range *payloads {
		if payload.VisitorIDScope != want[i] {
			t.Errorf("event %d visitorIdScope = %q, want %q", i, payload.VisitorIDScope, want[i])
		}
	}
}

func TestAnUnknownVisitorIDScopeIsLeftOut(t *testing.T) {
	body, err := json.Marshal(collectPayload{VisitorID: "visitor"})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["visitorIdScope"]; ok {
		t.Errorf("payload carries visitorIdScope without a scope: %s", body)
	}
}
