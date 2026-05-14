// Package updatecheck queries GitHub Releases for the latest stable tossctl
// release and caches the result on disk so the CLI can surface "new version
// available" without hammering the API on every command.
//
// The cache TTL is 24h; network failures are silent (the goroutine just
// returns the previously cached value, or empty string on first run).
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRepoSlug = "JungHoonGhae/tossinvest-cli"
	defaultInterval = 24 * time.Hour
	fetchTimeout    = 3 * time.Second
)

type cacheEntry struct {
	LastCheckedAt time.Time `json:"last_checked_at"`
	LatestVersion string    `json:"latest_version"`
}

type Checker struct {
	cachePath  string
	httpClient *http.Client
	repoSlug   string
	interval   time.Duration
	now        func() time.Time
}

func New(cachePath string) *Checker {
	return &Checker{
		cachePath:  cachePath,
		httpClient: &http.Client{Timeout: fetchTimeout},
		repoSlug:   defaultRepoSlug,
		interval:   defaultInterval,
		now:        time.Now,
	}
}

// LatestStable returns the cached or freshly-fetched latest tag (no "v"
// prefix). Empty string means "no info available" — the caller should treat
// it as a no-op rather than an error.
func (c *Checker) LatestStable(ctx context.Context) string {
	entry, _ := c.readCache()
	if c.now().Sub(entry.LastCheckedAt) < c.interval && entry.LatestVersion != "" {
		return entry.LatestVersion
	}

	latest, err := c.fetch(ctx)
	if err != nil || latest == "" {
		return entry.LatestVersion
	}

	_ = c.writeCache(cacheEntry{LastCheckedAt: c.now(), LatestVersion: latest})
	return latest
}

func (c *Checker) fetch(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.repoSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var payload struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.Prerelease {
		return "", nil
	}
	return strings.TrimPrefix(payload.TagName, "v"), nil
}

func (c *Checker) readCache() (cacheEntry, error) {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return cacheEntry{}, err
	}
	var e cacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return cacheEntry{}, err
	}
	return e, nil
}

func (c *Checker) writeCache(e cacheEntry) error {
	if err := os.MkdirAll(filepath.Dir(c.cachePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.cachePath, data, 0o600)
}

// IsNewer reports whether `latest` is semantically newer than `current`.
// Both arguments are bare semver strings without the "v" prefix. Returns
// false for empty or non-numeric inputs (e.g. dev builds) — the caller is
// expected to treat that as "skip the notice."
func IsNewer(latest, current string) bool {
	if latest == "" || current == "" || current == "dev" {
		return false
	}
	return compareSemver(latest, current) > 0
}

func compareSemver(a, b string) int {
	aParts := splitSemver(a)
	bParts := splitSemver(b)
	for i := 0; i < 3; i++ {
		av, bv := 0, 0
		if i < len(aParts) {
			av = aParts[i]
		}
		if i < len(bParts) {
			bv = bParts[i]
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}

func splitSemver(s string) []int {
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, 3)
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out
		}
		out = append(out, n)
	}
	return out
}
