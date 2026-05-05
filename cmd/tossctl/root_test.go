package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func TestExpiryWarningWithin24Hours(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(18 * time.Hour)
	sess := &session.Session{ServerExpiresAt: &exp}

	var buf bytes.Buffer
	writeExpiryWarningIfNeeded(&buf, sess, "portfolio", output.FormatTable, time.Now())
	got := buf.String()
	if !strings.Contains(got, "session expires") {
		t.Fatalf("expected warning, got %q", got)
	}
	if !strings.Contains(got, "tossctl auth extend") {
		t.Fatalf("expected hint about auth extend, got %q", got)
	}
}

func TestExpiryWarningSilentBeyond24Hours(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(48 * time.Hour)
	sess := &session.Session{ServerExpiresAt: &exp}

	var buf bytes.Buffer
	writeExpiryWarningIfNeeded(&buf, sess, "portfolio", output.FormatTable, time.Now())
	if buf.Len() != 0 {
		t.Fatalf("expected silence, got %q", buf.String())
	}
}

func TestExpiryWarningSilentInJSONMode(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(2 * time.Hour)
	sess := &session.Session{ServerExpiresAt: &exp}

	var buf bytes.Buffer
	writeExpiryWarningIfNeeded(&buf, sess, "portfolio", output.FormatJSON, time.Now())
	if buf.Len() != 0 {
		t.Fatalf("expected silence in JSON mode, got %q", buf.String())
	}
}

func TestExpiryWarningSilentForExtendCommand(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(2 * time.Hour)
	sess := &session.Session{ServerExpiresAt: &exp}

	for _, name := range []string{"extend", "login", "logout", "status", "import-playwright-state", "version", "help"} {
		var buf bytes.Buffer
		writeExpiryWarningIfNeeded(&buf, sess, name, output.FormatTable, time.Now())
		if buf.Len() != 0 {
			t.Fatalf("expected silence for %q, got %q", name, buf.String())
		}
	}
}

func TestExpiryWarningSilentWhenAlreadyExpired(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(-1 * time.Hour)
	sess := &session.Session{ServerExpiresAt: &exp}

	var buf bytes.Buffer
	writeExpiryWarningIfNeeded(&buf, sess, "portfolio", output.FormatTable, time.Now())
	// Already expired — let the 401 path handle it; don't add noise.
	if buf.Len() != 0 {
		t.Fatalf("expected silence when already expired, got %q", buf.String())
	}
}
