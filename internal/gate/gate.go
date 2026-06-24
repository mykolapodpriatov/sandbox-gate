// Package gate is the orchestration core: it merges CLI flags, the optional
// config file, and stack auto-detection into a concrete sandbox.Spec, selects a
// backend, runs it, and maps the result to a CI exit code. Resolve is pure and
// unit-tested; Run delegates to the backend.
package gate

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/mykolapodpriatov/sandbox-gate/internal/config"
	"github.com/mykolapodpriatov/sandbox-gate/internal/detect"
	"github.com/mykolapodpriatov/sandbox-gate/internal/sandbox"
)

// Secure-by-default isolation knobs (used when neither flag nor config sets them).
const (
	defMemoryMB  = 1024
	defCPUs      = 2.0
	defPidsLimit = 512
	defTimeout   = 5 * time.Minute
)

// Inputs are the raw CLI flags. Empty strings / zero numbers mean "unset";
// pointers mean "explicitly set" (so --net=false can override a config true).
type Inputs struct {
	Dir        string
	Image      string
	Cmd        string
	Backend    string
	Network    *bool
	MemoryMB   int
	CPUs       float64
	PidsLimit  int
	NonRoot    *bool
	Timeout    time.Duration
	ConfigPath string
}

// Resolve merges, in precedence order, flag > config file > auto-detection >
// secure default, into a Spec and a backend name.
func Resolve(in Inputs) (spec sandbox.Spec, backend string, err error) {
	cfg, err := config.Load(in.ConfigPath)
	if err != nil {
		return sandbox.Spec{}, "", fmt.Errorf("config: %w", err)
	}

	dir := firstStr(in.Dir, ".")
	image := firstStr(in.Image, cfg.Image)
	cmd := firstStr(in.Cmd, cfg.Cmd)
	if image == "" || cmd == "" {
		if st, ok := detect.Detect(dir); ok {
			image = firstStr(image, st.Image)
			cmd = firstStr(cmd, st.Cmd)
		}
	}
	if image == "" {
		return sandbox.Spec{}, "", fmt.Errorf("no base image: pass --image or set it in .sandbox-gate.yml (could not auto-detect the stack)")
	}
	if cmd == "" {
		return sandbox.Spec{}, "", fmt.Errorf("no test command: pass --cmd or set it in .sandbox-gate.yml")
	}

	cfgTimeout, err := cfg.ParseTimeout()
	if err != nil {
		return sandbox.Spec{}, "", fmt.Errorf("config timeout: %w", err)
	}

	spec = sandbox.Spec{
		SourceDir: dir,
		Image:     image,
		Cmd:       cmd,
		Timeout:   firstDur(in.Timeout, cfgTimeout, defTimeout),
		Isolation: sandbox.Isolation{
			Network:   boolPref(in.Network, cfg.Network, false), // no egress by default
			MemoryMB:  firstInt(in.MemoryMB, cfg.MemoryMB, defMemoryMB),
			CPUs:      firstFloat(in.CPUs, cfg.CPUs, defCPUs),
			PidsLimit: firstInt(in.PidsLimit, cfg.PidsLimit, defPidsLimit),
			NonRoot:   boolPref(in.NonRoot, cfg.NonRoot, true), // unprivileged by default
		},
	}
	return spec, firstStr(in.Backend, cfg.Backend, "docker"), nil
}

// Select returns the backend for a name. docker is the supported driver; e2b and
// daytona are recognized names that implement sandbox.Backend as the next drivers.
func Select(name string) (sandbox.Backend, error) {
	switch name {
	case "", "docker":
		return sandbox.Docker{}, nil
	case "e2b", "daytona":
		return nil, fmt.Errorf("backend %q is not compiled into this release — docker is the supported backend; %q is the documented next driver (implement sandbox.Backend)", name, name)
	default:
		return nil, fmt.Errorf("unknown backend %q (supported: docker)", name)
	}
}

// Run executes the spec on the backend, streaming logs to out.
func Run(ctx context.Context, b sandbox.Backend, spec sandbox.Spec, out io.Writer) (sandbox.Result, error) {
	return b.Run(ctx, spec, out)
}

// ExitCode maps a result to the process exit code a CI gate expects.
func ExitCode(r sandbox.Result) int {
	switch {
	case r.Passed():
		return 0
	case r.Status == "timeout":
		return 124
	default:
		return 1
	}
}

func firstStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
func firstInt(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
func firstFloat(vals ...float64) float64 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
func firstDur(vals ...time.Duration) time.Duration {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
func boolPref(flag, cfg *bool, def bool) bool {
	if flag != nil {
		return *flag
	}
	if cfg != nil {
		return *cfg
	}
	return def
}
