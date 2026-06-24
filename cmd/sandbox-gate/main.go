// Command sandbox-gate runs an untrusted test suite inside an isolated, ephemeral
// sandbox and exits 0 (pass) or 1 (fail) — a drop-in CI gate for AI-generated code.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mykolapodpriatov/sandbox-gate/internal/gate"
	"github.com/mykolapodpriatov/sandbox-gate/internal/report"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		os.Exit(runCmd(os.Args[2:]))
	case "init":
		os.Exit(initCmd(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println("sandbox-gate", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runCmd(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dir := fs.String("dir", ".", "worktree to validate")
	image := fs.String("image", "", "base image (auto-detected from the stack if empty)")
	cmd := fs.String("cmd", "", "test command (auto-detected from the stack if empty)")
	backend := fs.String("backend", "", "sandbox backend (docker)")
	net := fs.Bool("net", false, "allow network egress inside the sandbox")
	nonRoot := fs.Bool("non-root", true, "run the suite as an unprivileged uid")
	memory := fs.Int("memory", 0, "memory limit in MB (default 1024)")
	cpus := fs.Float64("cpus", 0, "CPU limit (default 2)")
	pids := fs.Int("pids", 0, "max process count (default 512)")
	timeout := fs.Duration("timeout", 0, "wall-clock timeout (default 5m)")
	cfgPath := fs.String("config", "", "config file (default .sandbox-gate.yml)")
	out := fs.String("out", "", "write the JSON result to this path")
	_ = fs.Parse(args)

	// Distinguish "flag explicitly set" from "left at default" so --net / --non-root
	// can override a config-file value rather than always winning.
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	var netP, nonRootP *bool
	if set["net"] {
		netP = net
	}
	if set["non-root"] {
		nonRootP = nonRoot
	}

	spec, backendName, err := gate.Resolve(gate.Inputs{
		Dir: *dir, Image: *image, Cmd: *cmd, Backend: *backend,
		Network: netP, NonRoot: nonRootP,
		MemoryMB: *memory, CPUs: *cpus, PidsLimit: *pids,
		Timeout: *timeout, ConfigPath: *cfgPath,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sandbox-gate:", err)
		return 2
	}
	b, err := gate.Select(backendName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sandbox-gate:", err)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(os.Stderr, "→ sandbox-gate: `%s` in %s  (net=%v, non-root=%v, %dMB, %g cpu, backend=%s)\n",
		spec.Cmd, spec.Image, spec.Isolation.Network, spec.Isolation.NonRoot,
		spec.Isolation.MemoryMB, spec.Isolation.CPUs, b.Name())

	res, _ := gate.Run(ctx, b, spec, os.Stdout)
	report.Summary(os.Stderr, res)
	if err := report.WriteJSON(*out, res); err != nil {
		fmt.Fprintln(os.Stderr, "sandbox-gate: write result:", err)
	}
	return gate.ExitCode(res)
}

func initCmd(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	ci := fs.String("ci", "", "emit a CI snippet instead of a config: github | gitlab")
	_ = fs.Parse(args)
	switch *ci {
	case "":
		fmt.Print(sampleConfig)
	case "github":
		fmt.Print(githubSnippet)
	case "gitlab":
		fmt.Print(gitlabSnippet)
	default:
		fmt.Fprintf(os.Stderr, "unknown --ci %q (github|gitlab)\n", *ci)
		return 2
	}
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `sandbox-gate — run an untrusted test suite in an isolated sandbox; exit 0=pass, 1=fail.

USAGE:
  sandbox-gate run   [flags]      validate a worktree in a sandbox
  sandbox-gate init  [--ci X]     print a sample config or a CI snippet
  sandbox-gate version

RUN FLAGS:
  --dir PATH        worktree to validate (default ".")
  --image IMG       base image (auto-detected if omitted)
  --cmd "CMD"       test command (auto-detected if omitted)
  --net             allow network egress (default: no egress)
  --memory MB       memory limit (default 1024)
  --cpus N          cpu limit (default 2)
  --pids N          max processes (default 512)
  --timeout DUR     wall-clock cap (default 5m)
  --out FILE        write JSON result

EXAMPLES:
  sandbox-gate run --dir ./agent-output
  sandbox-gate run --image node:20-alpine --cmd "npm test" --out result.json
  sandbox-gate run --net --timeout 2m       # opt back into network, shorter cap
`)
}

const sampleConfig = `# .sandbox-gate.yml — all fields optional
image: node:20-alpine
cmd: npm test
backend: docker
network: false      # no egress for the suite
memory_mb: 1024
cpus: 2
pids_limit: 512
non_root: true
timeout: 5m
`

const githubSnippet = `# .github/workflows/sandbox-gate.yml
name: sandbox-gate
on: [pull_request]
jobs:
  gate:
    runs-on: ubuntu-latest   # has Docker preinstalled
    steps:
      - uses: actions/checkout@v4
      - uses: mykolapodpriatov/sandbox-gate@v1
        with:
          dir: .
          out: result.json
`

const gitlabSnippet = `# .gitlab-ci.yml
sandbox-gate:
  image: docker:28
  services: [docker:28-dind]
  script:
    - go run github.com/mykolapodpriatov/sandbox-gate/cmd/sandbox-gate@latest run --dir .
`
