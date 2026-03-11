package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func TestListWatchlistFromFixtures(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/dashboard/asset/sections/all":
			http.ServeFile(w, r, watchlistFixturePath(t))
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

	items, err := client.ListWatchlist(context.Background())
	if err != nil {
		t.Fatalf("ListWatchlist returned error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one watchlist item")
	}
	if items[0].Symbol == "" {
		t.Fatal("expected first watchlist item to have a symbol")
	}
}

func watchlistFixturePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "auth-sanitized", "asset-sections-v2.json")
}
