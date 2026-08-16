package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsWhitespaceText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"--text", " \t\n "}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("run() exit code = %d, want 2", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--text is required") {
		t.Fatalf("stderr %q did not mention required text", stderr.String())
	}
}

func TestRunSupportsChunkingFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{
		"--text", "First part. Second part. Third part.",
		"--max-words", "2",
		"--max-chars", "20",
		"--longform-threshold", "1",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, stderr = %q", exitCode, stderr.String())
	}

	got := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	want := []string{"First part.", "Second part.", "Third part."}
	if len(got) != len(want) {
		t.Fatalf("got %d chunks %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunk %d = %q, want %q", i, got[i], want[i])
		}
	}
}
