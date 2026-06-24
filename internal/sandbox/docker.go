package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// Docker runs the suite in a throwaway container via the local `docker` CLI. It
// is the default backend: it needs no API keys and works on any dev machine or
// CI runner with a Docker daemon.
//
// Isolation is enforced with daemon-level controls, not just convention:
//   - the worktree is bind-mounted READ-ONLY at /src and copied into a tmpfs
//     /work, so the suite cannot mutate the host checkout;
//   - --network=none by default — AI-generated code gets no egress unless the
//     caller explicitly opts in;
//   - cgroup memory/cpu/pids limits bound a runaway or fork-bomb suite;
//   - the container runs as an unprivileged uid and the host Docker socket is
//     never mounted, so a compromised suite cannot reach the daemon.
type Docker struct {
	Bin string // docker binary; "" → "docker"
}

func (d Docker) Name() string { return "docker" }

func (d Docker) bin() string {
	if d.Bin != "" {
		return d.Bin
	}
	return "docker"
}

func (d Docker) Run(ctx context.Context, spec Spec, out io.Writer) (Result, error) {
	start := time.Now()
	res := Result{Backend: d.Name(), Image: spec.Image, Isolation: spec.Isolation}

	src, err := filepath.Abs(spec.SourceDir)
	if err != nil {
		return res, fmt.Errorf("resolve source dir: %w", err)
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if spec.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}

	args := []string{"run", "--rm", "--workdir", "/work"}
	if !spec.Isolation.Network {
		args = append(args, "--network", "none")
	}
	if spec.Isolation.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", spec.Isolation.MemoryMB))
	}
	if spec.Isolation.CPUs > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(spec.Isolation.CPUs, 'g', -1, 64))
	}
	if spec.Isolation.PidsLimit > 0 {
		args = append(args, "--pids-limit", strconv.Itoa(spec.Isolation.PidsLimit))
	}
	if spec.Isolation.NonRoot {
		args = append(args, "--user", "1000:1000")
	}
	args = append(args,
		"--mount", fmt.Sprintf("type=bind,src=%s,dst=/src,ro", src),
		"--tmpfs", "/work:exec",
		spec.Image, "sh", "-c", script(spec.Cmd),
	)

	cmd := exec.CommandContext(runCtx, d.bin(), args...)
	cmd.Stdout = out
	cmd.Stderr = out
	runErr := cmd.Run()
	res.DurationMS = time.Since(start).Milliseconds()

	switch {
	case runCtx.Err() == context.DeadlineExceeded:
		res.Status, res.ExitCode = "timeout", 124
		return res, nil
	case runErr == nil:
		res.Status, res.ExitCode = "pass", 0
		return res, nil
	default:
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			// A non-zero exit from the suite (or a 125/126/127 from docker) — the
			// gate treats a docker launch error specially so it is not silently
			// reported as a test "fail".
			if ee.ExitCode() >= 125 && ee.ExitCode() <= 127 {
				res.Status, res.ExitCode, res.Err = "error", ee.ExitCode(), "docker could not start the container (image/exec error)"
				return res, fmt.Errorf("docker run failed: exit %d", ee.ExitCode())
			}
			res.Status, res.ExitCode = "fail", ee.ExitCode()
			return res, nil
		}
		// docker binary missing, etc.
		res.Status, res.ExitCode, res.Err = "error", -1, runErr.Error()
		return res, runErr
	}
}

// script copies the read-only source into the writable workdir and runs the test
// command there. Keeping the copy inside the container (rather than a writable
// bind mount) is what lets the host checkout stay read-only.
func script(cmd string) string {
	return "cp -a /src/. /work/ 2>/dev/null; cd /work && " + cmd
}
