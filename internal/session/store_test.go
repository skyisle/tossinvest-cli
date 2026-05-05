package session

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionServerExpiresAtRoundTrip(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "session.json")
	store := NewFileStore(path)

	cookie := time.Date(2027, 4, 28, 22, 3, 53, 0, time.UTC)
	server := time.Date(2026, 5, 13, 7, 3, 20, 0, time.FixedZone("KST", 9*3600))

	in := &Session{
		Provider:        "playwright-storage-state",
		Cookies:         map[string]string{"SESSION": "x"},
		RetrievedAt:     time.Now().UTC(),
		ExpiresAt:       &cookie,
		ServerExpiresAt: &server,
	}
	if err := store.Save(context.Background(), in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.ServerExpiresAt == nil || !out.ServerExpiresAt.Equal(server) {
		t.Fatalf("ServerExpiresAt mismatch: got %v, want %v", out.ServerExpiresAt, &server)
	}
	if out.ExpiresAt == nil || !out.ExpiresAt.Equal(cookie) {
		t.Fatalf("ExpiresAt mismatch")
	}
}
