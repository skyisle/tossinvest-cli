package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/version"
)

// PostDiscord sends a Discord webhook message summarizing failed probes.
// No-op when webhookURL is empty or there are zero failures.
//
// Privacy: only probe names + status codes + truncated body samples (≤200B
// per probe) are sent. No cookies, account numbers, session tokens, or
// dollar amounts cross the webhook. The Toss session is used to fetch the
// data; the webhook only sees pass/fail metadata.
func PostDiscord(ctx context.Context, webhookURL string, results []Result) error {
	if webhookURL == "" {
		return nil
	}
	var failed []Result
	for _, r := range results {
		if !r.OK {
			failed = append(failed, r)
		}
	}
	if len(failed) == 0 {
		return nil
	}

	var b strings.Builder
	v := version.Current()
	fmt.Fprintf(&b, "🚨 **tossctl API regression detected** (`%s`)\n", v.Version)
	fmt.Fprintf(&b, "_%s — %d/%d probes failed_\n\n", time.Now().UTC().Format("2006-01-02 15:04 UTC"), len(failed), len(results))
	for _, r := range failed {
		fmt.Fprintf(&b, "❌ **%s** — `%s %s`\n", r.Probe.Name, r.Probe.Method, shortURL(r.Probe.URL))
		fmt.Fprintf(&b, "    status=%d, %s\n", r.Status, r.Detail)
	}
	// Discord limit: 2000 chars per content field.
	content := b.String()
	if len(content) > 1900 {
		content = content[:1900] + "...\n_(truncated)_"
	}

	payload, _ := json.Marshal(map[string]any{"content": content})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

// shortURL strips the wts-*.tossinvest.com prefix to keep Discord messages tight.
func shortURL(u string) string {
	for _, prefix := range []string{
		"https://wts-api.tossinvest.com",
		"https://wts-cert-api.tossinvest.com",
		"https://wts-info-api.tossinvest.com",
	} {
		if strings.HasPrefix(u, prefix) {
			return strings.TrimPrefix(u, "https://")
		}
	}
	return u
}
