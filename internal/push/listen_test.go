package push

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func TestParseStreamExtractsDataEvents(t *testing.T) {
	raw := strings.Join([]string{
		":heartbeat",
		"",
		"retry: 3600000",
		"",
		"id: abc123",
		`data: {"type":"pending-order-refresh","key":"1"}`,
		"",
		"id: def456",
		`data: {"type":"purchase-price-refresh","msg":{"stockCode":"US20000627001"},"key":"1"}`,
		"",
		"id: skip",
		"data: not-json",
		"",
	}, "\n")

	var got []Event
	if err := parseStream(strings.NewReader(raw), func(ev Event) {
		got = append(got, ev)
	}); err != nil {
		t.Fatalf("parseStream returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 valid events, got %d: %+v", len(got), got)
	}
	if got[0].ID != "abc123" || got[0].Type != "pending-order-refresh" {
		t.Fatalf("event 0 mismatch: %+v", got[0])
	}
	if got[1].Type != "purchase-price-refresh" || got[1].Msg["stockCode"] != "US20000627001" {
		t.Fatalf("event 1 mismatch: %+v", got[1])
	}
}

func TestListenRequiresSession(t *testing.T) {
	l := NewListener(&session.Session{})
	err := l.Listen(context.Background(), func(Event) {})
	if err != ErrNoSession {
		t.Fatalf("expected ErrNoSession, got %v", err)
	}
}

func TestListenSendsCookiesAndConsumesStream(t *testing.T) {
	var sawCookie atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("SESSION"); err == nil && c.Value == "sess-token" {
			sawCookie.Store(true)
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("missing Accept header: %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter is not a Flusher")
		}
		// One frame then close.
		_, _ = w.Write([]byte("id: 1\n"))
		_, _ = w.Write([]byte(`data: {"type":"pending-order-refresh","key":"1"}` + "\n\n"))
		flusher.Flush()
	}))
	defer srv.Close()

	l := NewListener(
		&session.Session{Cookies: map[string]string{"SESSION": "sess-token"}},
		WithStreamURL(srv.URL),
	)

	var events []Event
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := l.Listen(ctx, func(ev Event) { events = append(events, ev) }); err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}

	if !sawCookie.Load() {
		t.Fatal("expected SESSION cookie on request")
	}
	if len(events) != 1 || events[0].Type != "pending-order-refresh" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestListenRejectsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	l := NewListener(
		&session.Session{Cookies: map[string]string{"SESSION": "x"}},
		WithStreamURL(srv.URL),
	)
	err := l.Listen(context.Background(), func(Event) {})
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("expected HTTP 401 error, got %v", err)
	}
}
