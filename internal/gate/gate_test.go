package gate

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mykolapodpriatov/sandbox-gate/internal/sandbox"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolve_SecureDefaults(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\ngo 1.23\n")

	spec, backend, err := Resolve(Inputs{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if backend != "docker" {
		t.Errorf("backend = %q, want docker", backend)
	}
	if spec.Image != "golang:1.26-alpine" || spec.Cmd != "go test ./..." {
		t.Errorf("auto-detect failed: image=%q cmd=%q", spec.Image, spec.Cmd)
	}
	if spec.Isolation.Network {
		t.Error("network must default to OFF")
	}
	if !spec.Isolation.NonRoot {
		t.Error("non-root must default to ON")
	}
	if spec.Isolation.MemoryMB != defMemoryMB || spec.Timeout != defTimeout {
		t.Errorf("defaults: mem=%d timeout=%v", spec.Isolation.MemoryMB, spec.Timeout)
	}
}

func TestResolve_Precedence(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\ngo 1.23\n") // detection: golang image + go test
	cfg := filepath.Join(dir, ".sandbox-gate.yml")
	write(t, dir, ".sandbox-gate.yml", "image: node:20\ncmd: npm test\nmemory_mb: 256\n")

	// flag (image) beats config beats detection; config (cmd, mem) beats detection.
	spec, _, err := Resolve(Inputs{Dir: dir, ConfigPath: cfg, Image: "python:3.12"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Image != "python:3.12" {
		t.Errorf("flag should win for image: got %q", spec.Image)
	}
	if spec.Cmd != "npm test" {
		t.Errorf("config should win for cmd over detection: got %q", spec.Cmd)
	}
	if spec.Isolation.MemoryMB != 256 {
		t.Errorf("config should set memory: got %d", spec.Isolation.MemoryMB)
	}
}

func TestResolve_FlagOverridesConfigNetwork(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".sandbox-gate.yml")
	write(t, dir, ".sandbox-gate.yml", "image: alpine\ncmd: \"true\"\nnetwork: true\n")

	no := false
	spec, _, err := Resolve(Inputs{Dir: dir, ConfigPath: cfg, Network: &no})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Isolation.Network {
		t.Error("--net=false must override config network:true")
	}
}

func TestResolve_UnresolvableImageErrors(t *testing.T) {
	if _, _, err := Resolve(Inputs{Dir: t.TempDir()}); err == nil {
		t.Fatal("want an error when no image can be flagged, configured, or detected")
	}
}

func TestExitCode(t *testing.T) {
	for status, want := range map[string]int{"pass": 0, "fail": 1, "timeout": 124, "error": 1} {
		if got := ExitCode(sandbox.Result{Status: status}); got != want {
			t.Errorf("ExitCode(%q) = %d, want %d", status, got, want)
		}
	}
}

func TestSelect(t *testing.T) {
	if b, err := Select("docker"); err != nil || b.Name() != "docker" {
		t.Errorf("docker: %v %v", b, err)
	}
	if _, err := Select("e2b"); err == nil {
		t.Error("e2b should report it is not compiled in")
	}
	if _, err := Select("nope"); err == nil {
		t.Error("unknown backend should error")
	}
}

func TestRun_DelegatesToBackend(t *testing.T) {
	f := &sandbox.Fake{Result: sandbox.Result{Status: "pass"}}
	res, err := Run(context.Background(), f, sandbox.Spec{Image: "x", Cmd: "go test ./..."}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed() {
		t.Error("want pass from fake backend")
	}
	if f.Called != 1 || f.Got.Cmd != "go test ./..." {
		t.Errorf("backend not invoked with spec: called=%d got=%+v", f.Called, f.Got)
	}
}
