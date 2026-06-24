package report

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mykolapodpriatov/sandbox-gate/internal/sandbox"
)

func TestWriteJSON_RoundTrips(t *testing.T) {
	p := filepath.Join(t.TempDir(), "result.json")
	want := sandbox.Result{Status: "fail", ExitCode: 2, Backend: "docker", DurationMS: 42}
	if err := WriteJSON(p, want); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var got sandbox.Result
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "fail" || got.ExitCode != 2 || got.Backend != "docker" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestWriteJSON_NoPathIsNoop(t *testing.T) {
	if err := WriteJSON("", sandbox.Result{}); err != nil {
		t.Errorf("empty path should be a no-op, got %v", err)
	}
}

func TestSummary_MentionsStatus(t *testing.T) {
	var buf bytes.Buffer
	Summary(&buf, sandbox.Result{Status: "pass", ExitCode: 0, Isolation: sandbox.Isolation{Network: false}})
	out := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("pass")) || !bytes.Contains(buf.Bytes(), []byte("net=off")) {
		t.Errorf("summary missing fields: %q", out)
	}
}
