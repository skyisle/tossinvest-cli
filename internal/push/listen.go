// Package push subscribes to the Toss Securities SSE notification channel.
//
// Toss exposes a single long-lived HTTP GET stream at
//   https://sse-message.tossinvest.com/api/v1/wts-notification
// delivering thin "something changed, re-fetch" notifications.
// See docs/reverse-engineering/push-events.md for the event taxonomy.
package push

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

const (
	defaultStreamURL     = "https://sse-message.tossinvest.com/api/v1/wts-notification"
	defaultUserAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
	initialBackoff       = 2 * time.Second
	maxBackoff           = 60 * time.Second
	readDeadlineInterval = 0 // 0 = no read deadline; the server's heartbeats keep the connection warm
)

// ErrNoSession is returned when the listener is started without valid session cookies.
var ErrNoSession = errors.New("push: no active session cookies")

// Event is a parsed SSE frame with its JSON data payload.
type Event struct {
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type"`
	Msg      map[string]any `json:"msg,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`      // full data payload as JSON
	Received time.Time      `json:"received"`
}

// Listener holds the dependencies for a single SSE subscription.
type Listener struct {
	session    *session.Session
	streamURL  string
	httpClient *http.Client
	logf       func(format string, args ...any)
}

// Option customises a Listener.
type Option func(*Listener)

// WithStreamURL overrides the SSE endpoint (mostly for tests).
func WithStreamURL(u string) Option {
	return func(l *Listener) { l.streamURL = u }
}

// WithHTTPClient swaps the underlying HTTP client. Pass a client without
// Timeout so the stream can stay open.
func WithHTTPClient(c *http.Client) Option {
	return func(l *Listener) { l.httpClient = c }
}

// WithLogger routes informational logs (reconnects, backoff) somewhere
// other than /dev/null.
func WithLogger(logf func(string, ...any)) Option {
	return func(l *Listener) { l.logf = logf }
}

// NewListener constructs a Listener using the session cookies from sess.
func NewListener(sess *session.Session, opts ...Option) *Listener {
	l := &Listener{
		session:    sess,
		streamURL:  defaultStreamURL,
		httpClient: &http.Client{}, // no Timeout: SSE is long-lived
		logf:       func(string, ...any) {},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Listen opens the SSE stream once and calls handler for every parsed event.
// It returns when ctx is cancelled or the connection ends. Retry logic is
// the caller's responsibility; see ListenWithRetry.
func (l *Listener) Listen(ctx context.Context, handler func(Event)) error {
	if l.session == nil || len(l.session.Cookies) == 0 {
		return ErrNoSession
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.streamURL, nil)
	if err != nil {
		return fmt.Errorf("push: build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", "https://www.tossinvest.com/")
	req.Header.Set("Origin", "https://www.tossinvest.com")
	for name, value := range l.session.Cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("push: open stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("push: stream returned HTTP %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		return fmt.Errorf("push: unexpected content-type %q", ct)
	}

	return parseStream(resp.Body, handler)
}

// ListenWithRetry wraps Listen in a reconnect loop with exponential backoff.
// It returns only when ctx is cancelled, or when a fatal error (e.g.
// missing session) is encountered.
func (l *Listener) ListenWithRetry(ctx context.Context, handler func(Event)) error {
	backoff := initialBackoff
	for {
		err := l.Listen(ctx, handler)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, ErrNoSession) {
			return err
		}
		if err != nil {
			l.logf("push: stream error, reconnecting in %s: %v", backoff, err)
		} else {
			l.logf("push: stream closed by server, reconnecting in %s", backoff)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// parseStream reads newline-delimited SSE lines from r, emitting one Event
// per blank-line-terminated frame that has a parseable `data:` JSON object.
// Frames without data (heartbeats, retry directives) are silently dropped.
func parseStream(r io.Reader, handler func(Event)) error {
	scanner := bufio.NewScanner(r)
	// Default scanner buffer is 64KB; Toss events are tiny (<200B observed),
	// but bump the max to 1MB just in case a future event type is larger.
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	var (
		id       string
		dataBuf  strings.Builder
		hasData  bool
	)
	flush := func() {
		defer func() {
			id = ""
			dataBuf.Reset()
			hasData = false
		}()
		if !hasData {
			return
		}
		payload := dataBuf.String()
		var parsed map[string]any
		if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
			// Non-JSON data frame — skip silently.
			return
		}
		ev := Event{
			ID:       id,
			Received: time.Now().UTC(),
			Raw:      parsed,
		}
		if t, ok := parsed["type"].(string); ok {
			ev.Type = t
		}
		if m, ok := parsed["msg"].(map[string]any); ok {
			ev.Msg = m
		}
		handler(ev)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, ":") {
			// SSE comment line (used for Toss heartbeats).
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if !found {
			// Field name with no value ("event\n") — treat whole line as field, empty value.
			field = line
			value = ""
		}
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "id":
			id = value
		case "data":
			if hasData {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(value)
			hasData = true
		case "retry", "event":
			// `retry` is a reconnect hint (unused here — we manage backoff).
			// `event` is the named event type; Toss uses the `type` field
			// inside `data` instead, so ignore this too.
		}
	}

	// Trailing frame without blank-line terminator (rare, but SSE allows it).
	flush()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("push: read stream: %w", err)
	}
	return nil
}
