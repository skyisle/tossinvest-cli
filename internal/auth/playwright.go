package auth

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

type playwrightStorageState struct {
	Cookies []struct {
		Name    string  `json:"name"`
		Value   string  `json:"value"`
		Expires float64 `json:"expires"`
	} `json:"cookies"`
	Origins []struct {
		Origin       string `json:"origin"`
		LocalStorage []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"localStorage"`
	} `json:"origins"`
}

func (s *Service) ImportPlaywrightState(ctx context.Context, path string) (*session.Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state playwrightStorageState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	sess := &session.Session{
		Provider:    "playwright-storage-state",
		Cookies:     map[string]string{},
		Headers:     map[string]string{},
		Storage:     map[string]string{},
		RetrievedAt: time.Now().UTC(),
	}

	for _, cookie := range state.Cookies {
		sess.Cookies[cookie.Name] = cookie.Value
		if cookie.Name == "SESSION" && cookie.Expires > 0 {
			expiresAt := time.Unix(int64(cookie.Expires), 0).UTC()
			sess.ExpiresAt = &expiresAt
		}
	}

	for _, origin := range state.Origins {
		if origin.Origin != "https://www.tossinvest.com" {
			continue
		}
		for _, item := range origin.LocalStorage {
			sess.Storage["localStorage:"+item.Name] = item.Value
		}
	}

	if token := sess.Cookies["XSRF-TOKEN"]; token != "" {
		sess.Headers["X-XSRF-TOKEN"] = token
	}
	if browserTabID := sess.Storage["localStorage:qr-tabId"]; browserTabID != "" {
		sess.Headers["Browser-Tab-Id"] = browserTabID
	}

	if err := s.store.Save(ctx, sess); err != nil {
		return nil, err
	}

	return sess, nil
}
