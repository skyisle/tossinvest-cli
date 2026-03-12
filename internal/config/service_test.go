package config

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStatusFallsBackToDefaultWhenConfigIsMissing(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.json"))

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Exists {
		t.Fatal("expected config to be absent")
	}
	if status.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %d, got %d", SchemaVersion, status.SchemaVersion)
	}
	if status.Trading.Place {
		t.Fatal("expected place to be disabled by default")
	}
}

func TestInitCreatesDefaultConfig(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.json"))

	result, err := service.Init(context.Background())
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if !result.Created {
		t.Fatal("expected config file to be created")
	}
	if !result.Status.Exists {
		t.Fatal("expected config file to exist after init")
	}
	if result.Status.Schema != DefaultSchemaURL {
		t.Fatalf("expected schema url %q, got %q", DefaultSchemaURL, result.Status.Schema)
	}
}
