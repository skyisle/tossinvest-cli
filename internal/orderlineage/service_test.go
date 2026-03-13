package orderlineage

import (
	"path/filepath"
	"testing"
)

func TestRecordAndResolveFollowRolloverChain(t *testing.T) {
	t.Parallel()

	service := NewService(filepath.Join(t.TempDir(), "lineage.json"))
	if err := service.Record("2026-03-13/1", "2026-03-13/2", "amend"); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := service.Record("2026-03-13/2", "2026-03-13/3", "cancel"); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	resolved, ok, err := service.Resolve("2026-03-13/1")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected lineage resolution")
	}
	if resolved != "2026-03-13/3" {
		t.Fatalf("expected final order id 2026-03-13/3, got %q", resolved)
	}
}

func TestResolveReturnsNoAliasWhenUnchanged(t *testing.T) {
	t.Parallel()

	service := NewService(filepath.Join(t.TempDir(), "lineage.json"))
	resolved, ok, err := service.Resolve("2026-03-13/1")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected no alias, got %q", resolved)
	}
}
