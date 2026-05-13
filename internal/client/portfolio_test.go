package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func TestListPositionsFromFixtures(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/dashboard/asset/sections/all":
			http.ServeFile(w, r, portfolioFixturePath(t))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session: &session.Session{
			Cookies: map[string]string{"SESSION": "test-session"},
		},
	})

	positions, err := client.ListPositions(context.Background())
	if err != nil {
		t.Fatalf("ListPositions returned error: %v", err)
	}
	if len(positions) == 0 {
		t.Fatal("expected at least one position")
	}
	if positions[0].Name == "" {
		t.Fatal("expected first position to have a name")
	}
}

// Issue #29: Toss server (2026-05-13) started ignoring the old empty `{}`
// body and requires an explicit `types` filter. Without the filter, sections
// comes back empty and ListPositions raised "SORTED_OVERVIEW section not
// found". Assert the wire body keeps the filter going forward.
func TestListPositionsSendsTypesFilter(t *testing.T) {
	t.Parallel()

	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/dashboard/asset/sections/all" {
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
		}
		http.ServeFile(w, r, portfolioFixturePath(t))
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		InfoBaseURL: server.URL,
		CertBaseURL: server.URL,
		Session:     &session.Session{Cookies: map[string]string{"SESSION": "test-session"}},
	})

	if _, err := client.ListPositions(context.Background()); err != nil {
		t.Fatalf("ListPositions returned error: %v", err)
	}
	if !strings.Contains(capturedBody, `"types"`) || !strings.Contains(capturedBody, `"SORTED_OVERVIEW"`) {
		t.Fatalf("expected wire body to carry types filter for SORTED_OVERVIEW, got %q", capturedBody)
	}
}

func portfolioFixturePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "auth-sanitized", "asset-sections-v2.json")
}
