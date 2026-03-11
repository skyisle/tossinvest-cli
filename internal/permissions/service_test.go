package permissions

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestGrantStatusAndRequire(t *testing.T) {
	dir := t.TempDir()
	service := NewService(filepath.Join(dir, "permission.json"))

	now := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	status, err := service.Grant(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}
	if !status.Active {
		t.Fatal("expected active permission after grant")
	}

	if err := service.Require(context.Background()); err != nil {
		t.Fatalf("Require returned error: %v", err)
	}

	service.now = func() time.Time { return now.Add(6 * time.Minute) }
	if err := service.Require(context.Background()); err != ErrExpiredGrant {
		t.Fatalf("expected ErrExpiredGrant, got %v", err)
	}
}

func TestRevokeWithoutExistingGrant(t *testing.T) {
	dir := t.TempDir()
	service := NewService(filepath.Join(dir, "permission.json"))

	cleared, err := service.Revoke(context.Background())
	if err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}
	if cleared {
		t.Fatal("expected no revoke when file was absent")
	}
}
