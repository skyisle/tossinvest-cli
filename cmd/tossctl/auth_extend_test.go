package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/auth"
)


func TestRunAuthExtendSucceeds(t *testing.T) {
	t.Parallel()

	expiry := time.Date(2026, 5, 13, 7, 3, 20, 0, time.FixedZone("KST", 9*3600))
	result := &auth.ExtendResult{
		UUID:            "abc",
		ServerExpiresAt: expiry,
		Elapsed:         3 * time.Second,
	}

	var out bytes.Buffer
	if err := writeExtendResult(&out, result); err != nil {
		t.Fatalf("writeExtendResult: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Extension complete") {
		t.Fatalf("expected success line, got %q", got)
	}
	if !strings.Contains(got, "2026-05-13 07:03 KST") {
		t.Fatalf("expected formatted KST expiry, got %q", got)
	}
}

func TestRunAuthExtendTimeoutMessage(t *testing.T) {
	t.Parallel()

	got := userFacingCommandError(auth.ErrExtensionTimeout).Error()
	if !strings.Contains(got, "rerun") {
		t.Fatalf("expected retry hint, got %q", got)
	}
	if !strings.Contains(got, "tossctl auth extend") {
		t.Fatalf("expected extend command hint, got %q", got)
	}
}

func TestRunAuthExtendRejectionMessage(t *testing.T) {
	t.Parallel()

	got := userFacingCommandError(auth.ErrExtensionRejected).Error()
	if !strings.Contains(got, "denied") && !strings.Contains(got, "canceled") {
		t.Fatalf("expected rejection wording, got %q", got)
	}
}

func TestWriteExtendResultRendersKSTForRFC3339ParsedInput(t *testing.T) {
	t.Parallel()

	// Simulate the production path: server returns RFC3339Nano with numeric
	// offset, parsed via time.Parse — the resulting Location is an anonymous
	// FixedZone with empty name. writeExtendResult must still display "KST".
	parsed, err := time.Parse(time.RFC3339Nano, "2026-05-13T07:03:20.24+09:00")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := &auth.ExtendResult{
		UUID:            "abc",
		ServerExpiresAt: parsed,
		Elapsed:         3 * time.Second,
	}

	var out bytes.Buffer
	if err := writeExtendResult(&out, result); err != nil {
		t.Fatalf("writeExtendResult: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "2026-05-13 07:03 KST") {
		t.Fatalf("expected KST label even for RFC3339-parsed input, got %q", got)
	}
}

// Sanity check: spinner runs without crashing when stdout is not a TTY.
func TestSpinnerSilentInNonTTY(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	stop := startSpinner(&out, "test", false)
	time.Sleep(20 * time.Millisecond)
	stop()
	if out.Len() != 0 {
		t.Fatalf("expected no output in non-TTY mode, got %q", out.String())
	}
}
