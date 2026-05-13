// Package monitor runs schema-invariant probes against the read-only Toss
// endpoints the CLI depends on, so that breaking server-side changes (like
// the body-contract change in #29) are caught by a cron job before users
// hit them.
//
// The checks are intentionally narrow: status 200 + a single critical
// JSON path with the expected JSON type. Schema flexibility on non-critical
// fields is allowed so Toss adding new fields does not trip alerts.
package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tossclient "github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

// Probe describes one endpoint to validate.
type Probe struct {
	Name   string // short id for logs / Discord
	Method string // GET / POST
	URL    string // absolute URL
	Body   string // empty for GET; required for POST
	// Check returns nil on success, error string on failure.
	Check func(status int, body []byte) error
}

// Result of one probe execution.
type Result struct {
	Probe    Probe
	OK       bool
	Status   int
	Duration time.Duration
	Detail   string // failure detail; empty if OK
}

// Probes returns the read-only endpoints we monitor.
//
// Picked to cover one representative endpoint per CLI surface:
//   - account / summary / positions / watchlist / quote / pending-orders
//
// Each probe's Check is a schema invariant — the smallest assertion that
// catches a contract change like #29 without false-positiving on Toss
// adding/removing unrelated fields.
func Probes() []Probe {
	const (
		api  = "https://wts-api.tossinvest.com"
		cert = "https://wts-cert-api.tossinvest.com"
		info = "https://wts-info-api.tossinvest.com"
	)
	return []Probe{
		{
			Name:   "account-list",
			Method: "GET",
			URL:    api + "/api/v1/account/list",
			Check: func(status int, body []byte) error {
				if err := expectStatus(status, body, 200); err != nil {
					return err
				}
				return expectPath(body, "result.accountList", "array")
			},
		},
		{
			Name:   "account-summary-overview",
			Method: "GET",
			URL:    cert + "/api/v3/my-assets/summaries/markets/all/overview",
			Check: func(status int, body []byte) error {
				if err := expectStatus(status, body, 200); err != nil {
					return err
				}
				if err := expectPath(body, "result.overviewByMarket", "object"); err != nil {
					return err
				}
				return expectPath(body, "result.totalAssetAmount", "number")
			},
		},
		{
			// Catches the #29 regression: empty `{}` body returns
			// empty sections + pollIntervalMillis. Real sections array
			// must contain a SORTED_OVERVIEW entry with products[].
			Name:   "portfolio-positions",
			Method: "POST",
			URL:    cert + "/api/v2/dashboard/asset/sections/all",
			Body:   `{"types":["SORTED_OVERVIEW"]}`,
			Check: func(status int, body []byte) error {
				if err := expectStatus(status, body, 200); err != nil {
					return err
				}
				if err := expectPath(body, "result.sections", "array"); err != nil {
					return err
				}
				// Drill into first section: type must match filter.
				var env struct {
					Result struct {
						Sections []struct {
							Type string `json:"type"`
							Data struct {
								Products json.RawMessage `json:"products"`
							} `json:"data"`
						} `json:"sections"`
					} `json:"result"`
				}
				if err := json.Unmarshal(body, &env); err != nil {
					return fmt.Errorf("decode sections: %v", err)
				}
				if len(env.Result.Sections) == 0 {
					return fmt.Errorf("result.sections is empty — likely body-contract regression (#29-class)")
				}
				if env.Result.Sections[0].Type != "SORTED_OVERVIEW" {
					return fmt.Errorf("expected section[0].type=SORTED_OVERVIEW, got %q", env.Result.Sections[0].Type)
				}
				if !bytes.HasPrefix(bytes.TrimSpace(env.Result.Sections[0].Data.Products), []byte("[")) {
					return fmt.Errorf("section[0].data.products is not an array")
				}
				return nil
			},
		},
		{
			Name:   "watchlist",
			Method: "POST",
			URL:    cert + "/api/v2/dashboard/asset/sections/all",
			Body:   `{"types":["WATCHLIST"]}`,
			Check: func(status int, body []byte) error {
				if err := expectStatus(status, body, 200); err != nil {
					return err
				}
				var env struct {
					Result struct {
						Sections []struct {
							Type string `json:"type"`
						} `json:"sections"`
					} `json:"result"`
				}
				if err := json.Unmarshal(body, &env); err != nil {
					return fmt.Errorf("decode sections: %v", err)
				}
				if len(env.Result.Sections) == 0 {
					return fmt.Errorf("result.sections is empty — likely body-contract regression")
				}
				if env.Result.Sections[0].Type != "WATCHLIST" {
					return fmt.Errorf("expected section[0].type=WATCHLIST, got %q", env.Result.Sections[0].Type)
				}
				return nil
			},
		},
		{
			Name:   "quote-stock-infos",
			Method: "GET",
			URL:    info + "/api/v2/stock-infos/A005930",
			Check: func(status int, body []byte) error {
				if err := expectStatus(status, body, 200); err != nil {
					return err
				}
				if err := expectPath(body, "result.symbol", "string"); err != nil {
					return err
				}
				return expectPath(body, "result.currency", "string")
			},
		},
		{
			Name:   "pending-orders",
			Method: "GET",
			URL:    cert + "/api/v1/trading/orders/histories/all/pending",
			Check: func(status int, body []byte) error {
				if err := expectStatus(status, body, 200); err != nil {
					return err
				}
				return expectPath(body, "result", "array")
			},
		},
	}
}

// Run executes all probes sequentially using the session for auth.
func Run(ctx context.Context, sess *session.Session) []Result {
	results := make([]Result, 0, len(Probes()))
	for _, p := range Probes() {
		results = append(results, runOne(ctx, sess, p))
	}
	return results
}

func runOne(ctx context.Context, sess *session.Session, p Probe) Result {
	res := Result{Probe: p}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if p.Body != "" {
		bodyReader = strings.NewReader(p.Body)
	}
	req, err := http.NewRequestWithContext(reqCtx, p.Method, p.URL, bodyReader)
	if err != nil {
		res.Detail = "build request: " + err.Error()
		return res
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", tossclient.DefaultBrowserUserAgent)
	req.Header.Set("Referer", "https://www.tossinvest.com/")
	req.Header.Set("Origin", "https://www.tossinvest.com")
	if p.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if sess != nil {
		for k, v := range sess.Cookies {
			req.AddCookie(&http.Cookie{Name: k, Value: v})
		}
		for k, v := range sess.Headers {
			req.Header.Set(k, v)
		}
	}

	start := time.Now()
	resp, err := (&http.Client{}).Do(req)
	res.Duration = time.Since(start)
	if err != nil {
		res.Detail = "transport: " + err.Error()
		return res
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	res.Status = resp.StatusCode
	if checkErr := p.Check(resp.StatusCode, body); checkErr != nil {
		res.Detail = checkErr.Error()
		return res
	}
	res.OK = true
	return res
}

// expectStatus asserts the HTTP status.
//
// PRIVACY: the error message MUST NOT include the response body. Toss error
// payloads on 4xx/5xx routinely carry the account number, asset totals, or
// other PII; embedding even a 200-byte sample would forward those values to
// any configured Discord webhook. Status code + expected code alone is
// enough to alert — operators investigate the body locally with their own
// session.
func expectStatus(got int, body []byte, want int) error {
	if got == want {
		return nil
	}
	_ = body // intentionally unused — see PRIVACY note above.
	return fmt.Errorf("status %d (want %d)", got, want)
}

// expectPath walks a dotted JSON path (a.b.c) and asserts the value's type.
// Supported types: "string", "number", "bool", "object", "array", "null".
// Array indexing not supported — for nested-array checks, use a custom Check.
func expectPath(body []byte, path, wantType string) error {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("decode body: %v", err)
	}
	current := v
	for _, segment := range strings.Split(path, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q: expected object at %q, got %s", path, segment, jsonTypeOf(current))
		}
		next, found := obj[segment]
		if !found {
			return fmt.Errorf("path %q: key %q missing", path, segment)
		}
		current = next
	}
	got := jsonTypeOf(current)
	if got != wantType {
		return fmt.Errorf("path %q: expected %s, got %s", path, wantType, got)
	}
	return nil
}

func jsonTypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	}
	return "unknown"
}
