package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.4.13", "0.4.12", true},
		{"0.5.0", "0.4.99", true},
		{"1.0.0", "0.9.9", true},
		{"0.4.12", "0.4.12", false},
		{"0.4.11", "0.4.12", false},
		{"0.4.12", "dev", false},
		{"", "0.4.12", false},
		{"0.4.12", "", false},
		{"0.4.13-rc1", "0.4.12", true}, // prerelease suffix stripped, compare as 0.4.13
	}
	for _, c := range cases {
		got := IsNewer(c.latest, c.current)
		if got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestLatestStableHitsCacheWithinInterval(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	// Seed cache as if we just checked 1 minute ago.
	seed := cacheEntry{LastCheckedAt: time.Now().Add(-time.Minute), LatestVersion: "9.9.9"}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// HTTP server that fails the test if called — cache should win.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("HTTP server should not be hit when cache is fresh")
	}))
	defer server.Close()

	c := &Checker{
		cachePath:  cachePath,
		httpClient: server.Client(),
		repoSlug:   "x/y",
		interval:   24 * time.Hour,
		now:        time.Now,
	}

	if got := c.LatestStable(context.Background()); got != "9.9.9" {
		t.Errorf("expected cached value, got %q", got)
	}
}

func TestLatestStableRefreshesWhenStale(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	// Cache older than interval.
	seed := cacheEntry{LastCheckedAt: time.Now().Add(-48 * time.Hour), LatestVersion: "0.0.1"}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.4.99", "prerelease": false})
	}))
	defer server.Close()

	c := &Checker{
		cachePath:  cachePath,
		httpClient: server.Client(),
		repoSlug:   "x/y",
		interval:   24 * time.Hour,
		now:        time.Now,
	}
	// Override fetch URL via embedded test transport.
	c.httpClient = &http.Client{Transport: redirectTransport{base: server.URL}}

	if got := c.LatestStable(context.Background()); got != "0.4.99" {
		t.Errorf("expected refreshed value, got %q", got)
	}

	// Cache should be updated.
	var refreshed cacheEntry
	raw, _ := os.ReadFile(cachePath)
	_ = json.Unmarshal(raw, &refreshed)
	if refreshed.LatestVersion != "0.4.99" {
		t.Errorf("cache not updated, got %q", refreshed.LatestVersion)
	}
}

func TestLatestStableNetworkFailureReturnsCachedValue(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")
	seed := cacheEntry{LastCheckedAt: time.Now().Add(-48 * time.Hour), LatestVersion: "0.4.10"}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	c := &Checker{
		cachePath:  cachePath,
		httpClient: &http.Client{Transport: failingTransport{}},
		repoSlug:   "x/y",
		interval:   24 * time.Hour,
		now:        time.Now,
	}
	if got := c.LatestStable(context.Background()); got != "0.4.10" {
		t.Errorf("expected stale cached value on network failure, got %q", got)
	}
}

// redirectTransport rewrites all requests to point at the test server.
type redirectTransport struct{ base string }

func (rt redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := http.NewRequest(req.Method, rt.base, nil)
	if err != nil {
		return nil, err
	}
	target.Header = req.Header
	return http.DefaultTransport.RoundTrip(target)
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, http.ErrHandlerTimeout
}

func TestShouldNotifyUpdateBackoff(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")
	seed := cacheEntry{LastCheckedAt: time.Now().Add(-time.Minute), LatestVersion: "0.4.13"}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	c := &Checker{cachePath: cachePath, repoSlug: "x/y", interval: 24 * time.Hour, now: time.Now}

	// First call: should notify.
	latest, ok := c.ShouldNotifyUpdate(context.Background(), "0.4.12")
	if !ok || latest != "0.4.13" {
		t.Fatalf("first call: expected notify=true latest=0.4.13, got notify=%v latest=%q", ok, latest)
	}
	c.MarkUpdateNotified()

	// Second call within the interval: should suppress.
	if _, ok := c.ShouldNotifyUpdate(context.Background(), "0.4.12"); ok {
		t.Fatal("second call within interval: expected notify=false (suppressed by backoff)")
	}

	// Same version: should not notify.
	if _, ok := c.ShouldNotifyUpdate(context.Background(), "0.4.13"); ok {
		t.Fatal("same version: expected notify=false")
	}
}

func TestShouldNotifyExpiryHourlyBackoff(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")
	c := &Checker{cachePath: cachePath, repoSlug: "x/y", interval: 24 * time.Hour, now: time.Now}

	// No cache yet: should notify.
	if !c.ShouldNotifyExpiry() {
		t.Fatal("first call: expected notify=true")
	}
	c.MarkExpiryNotified()

	// Immediate re-check: suppressed.
	if c.ShouldNotifyExpiry() {
		t.Fatal("immediate re-check: expected notify=false (suppressed within hour)")
	}

	// Backdate the cache by 2h and re-check: should notify again.
	entry, _ := c.readCache()
	entry.ExpiryNotifiedAt = time.Now().Add(-2 * time.Hour)
	if err := c.writeCache(entry); err != nil {
		t.Fatal(err)
	}
	if !c.ShouldNotifyExpiry() {
		t.Fatal("after 2h: expected notify=true")
	}
}
