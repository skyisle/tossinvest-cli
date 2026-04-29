// Package push subscribes to the Toss Securities SSE notification channel.
// See docs/reverse-engineering/push-events.md for the channel and event taxonomy.
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
	defaultStreamURL = "https://sse-message.tossinvest.com/api/v1/wts-notification"
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
	initialBackoff   = 2 * time.Second
	maxBackoff       = 60 * time.Second
)

var ErrNoSession = errors.New("push: no active session cookies")

// errConnectionClose signals Toss's graceful `event: connection-close` frame,
// emitted every few minutes. The retry loop reconnects immediately without
// growing the backoff so a routine hand-off is not treated as an outage.
var errConnectionClose = errors.New("push: server requested reconnect")

type Event struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Type     string         `json:"type"`
	Msg      map[string]any `json:"msg,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
	Received time.Time      `json:"received"`
}

type Listener struct {
	session    *session.Session
	streamURL  string
	httpClient *http.Client
	logf       func(format string, args ...any)
}

type Option func(*Listener)

func WithStreamURL(u string) Option {
	return func(l *Listener) { l.streamURL = u }
}

// WithHTTPClient swaps the underlying HTTP client. The client must have no
// Timeout because the SSE stream is long-lived.
func WithHTTPClient(c *http.Client) Option {
	return func(l *Listener) { l.httpClient = c }
}

func WithLogger(logf func(string, ...any)) Option {
	return func(l *Listener) { l.logf = logf }
}

func NewListener(sess *session.Session, opts ...Option) *Listener {
	l := &Listener{
		session:    sess,
		streamURL:  defaultStreamURL,
		httpClient: &http.Client{},
		logf:       func(string, ...any) {},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

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

		graceful := errors.Is(err, errConnectionClose)
		switch {
		case graceful:
			l.logf("push: server requested graceful reconnect")
		case err != nil:
			l.logf("push: stream error, reconnecting in %s: %v", backoff, err)
		default:
			l.logf("push: stream closed by server, reconnecting in %s", backoff)
		}

		if !graceful {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		backoff = initialBackoff
	}
}

func parseStream(r io.Reader, handler func(Event)) error {
	scanner := bufio.NewScanner(r)
	// Toss events are tiny (<400B observed); 1MB is defensive headroom.
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	var (
		id        string
		eventName string
		dataBuf   strings.Builder
		hasData   bool
		gotClose  bool
	)
	flush := func() {
		defer func() {
			id = ""
			eventName = ""
			dataBuf.Reset()
			hasData = false
		}()
		if eventName == "connection-close" {
			gotClose = true
			return
		}
		if !hasData {
			return
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(dataBuf.String()), &parsed); err != nil {
			return
		}
		ev := Event{
			ID:       id,
			Name:     eventName,
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
			if gotClose {
				return errConnectionClose
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if !found {
			field = line
			value = ""
		}
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "id":
			id = value
		case "event":
			eventName = value
		case "data":
			if hasData {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(value)
			hasData = true
		}
	}

	flush()
	if gotClose {
		return errConnectionClose
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("push: read stream: %w", err)
	}
	return nil
}
