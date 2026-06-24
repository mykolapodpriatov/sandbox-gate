// Package sandbox defines the backend abstraction used to run an untrusted test
// suite inside an isolated, ephemeral environment. The default backend is Docker;
// e2b and daytona drivers implement the same interface.
package sandbox

import (
	"context"
	"io"
	"time"
)

// Isolation is the security envelope applied to a sandbox run. The zero value is
// deliberately the *most* restrictive sane default (no network, non-root, limits),
// so forgetting to set a field never silently widens the blast radius.
type Isolation struct {
	Network   bool    `json:"network"`     // false = no egress (default)
	MemoryMB  int     `json:"memory_mb"`   // 0 = backend default
	CPUs      float64 `json:"cpus"`        // 0 = backend default
	PidsLimit int     `json:"pids_limit"`  // 0 = backend default
	NonRoot   bool    `json:"non_root"`    // run as an unprivileged uid
}

// Spec fully describes one sandbox run.
type Spec struct {
	SourceDir string        // host path to the worktree to validate
	Image     string        // base OCI image, e.g. "node:20-alpine"
	Cmd       string        // shell command to run, e.g. "npm test"
	Timeout   time.Duration // wall-clock cap; 0 = no cap
	Isolation Isolation
}

// Result is the structured outcome of a run. It is the JSON contract emitted by
// `--out` and consumed by CI.
type Result struct {
	Status     string    `json:"status"`      // pass | fail | timeout | error
	ExitCode   int       `json:"exit_code"`
	DurationMS int64     `json:"duration_ms"`
	Backend    string    `json:"backend"`
	Image      string    `json:"image"`
	Isolation  Isolation `json:"isolation"`
	Err        string    `json:"error,omitempty"`
}

// Passed reports whether the gate should let the code through.
func (r Result) Passed() bool { return r.Status == "pass" }

// Backend runs a Spec to completion, streaming combined stdout/stderr to out.
// Implementations must enforce spec.Isolation and spec.Timeout. A non-nil error
// is reserved for infrastructure failures (backend unreachable, image pull
// failure); a failing *test suite* is a normal Result with Status "fail".
type Backend interface {
	Name() string
	Run(ctx context.Context, spec Spec, out io.Writer) (Result, error)
}
