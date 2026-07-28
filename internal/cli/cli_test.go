package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	app := New(Identification{Version: "0.0.0", GitCommit: "abc", GitDescription: "test", BuildDate: "now"})
	var stdout, stderr bytes.Buffer
	if err := app.Run([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "traceart 0.0.0") {
		t.Errorf("expected version banner, got %q", stdout.String())
	}
}

func TestRenderSmoke(t *testing.T) {
	app := New(Identification{})
	var stdout, stderr bytes.Buffer
	if err := app.Run([]string{"--seed", "1", "--no-color", "--destination", "tokyo"}, &stdout, &stderr); err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "TRACEART") || !strings.Contains(out, "Tokyo") {
		t.Errorf("expected TRACEART/Tokyo in output, got:\n%s", out)
	}
}

func TestUsageOnBadFlag(t *testing.T) {
	app := New(Identification{})
	var stdout, stderr bytes.Buffer
	err := app.Run([]string{"--nope"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error on unknown flag")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("expected usage in stderr, got %q", stderr.String())
	}
}
