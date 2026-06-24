package sandbox

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func dockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}

func TestScript_CopiesAndRuns(t *testing.T) {
	got := script("go test ./...")
	if !strings.Contains(got, "/src") || !strings.Contains(got, "/work") || !strings.HasSuffix(got, "go test ./...") {
		t.Errorf("script() = %q", got)
	}
}

// Integration tests below need a Docker daemon; they self-skip without one.

func TestDocker_PassAndFail(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	d := Docker{}
	var buf bytes.Buffer

	pass, err := d.Run(context.Background(), Spec{SourceDir: ".", Image: "alpine", Cmd: "exit 0"}, &buf)
	if err != nil {
		t.Fatalf("pass run errored: %v", err)
	}
	if !pass.Passed() || pass.ExitCode != 0 {
		t.Errorf("want pass, got %+v", pass)
	}

	fail, err := d.Run(context.Background(), Spec{SourceDir: ".", Image: "alpine", Cmd: "exit 3"}, &buf)
	if err != nil {
		t.Fatalf("fail run errored: %v", err)
	}
	if fail.Status != "fail" || fail.ExitCode != 3 {
		t.Errorf("want fail/exit 3, got %+v", fail)
	}
}

func TestDocker_NetworkOffByDefault(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	// With Isolation.Network=false the container gets --network=none, so DNS fails.
	res, err := Docker{}.Run(context.Background(),
		Spec{SourceDir: ".", Image: "alpine", Cmd: "wget -q -T5 -O /dev/null http://example.com"},
		&bytes.Buffer{})
	if err != nil {
		t.Fatalf("run errored: %v", err)
	}
	if res.Passed() {
		t.Error("egress must be blocked when Isolation.Network is false")
	}
}

func TestDocker_Timeout(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	res, err := Docker{}.Run(context.Background(),
		Spec{SourceDir: ".", Image: "alpine", Cmd: "sleep 30", Timeout: 2 * time.Second},
		&bytes.Buffer{})
	if err != nil {
		t.Fatalf("run errored: %v", err)
	}
	if res.Status != "timeout" {
		t.Errorf("want timeout, got %+v", res)
	}
}
