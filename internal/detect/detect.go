// Package detect infers a sane default base image and test command from the
// files present in a project, so `sandbox-gate run` works with zero config on
// common stacks.
package detect

import (
	"os"
	"path/filepath"
)

// Stack is a detected toolchain with its default sandbox image and test command.
type Stack struct {
	Name  string
	Image string
	Cmd   string
}

// rules are evaluated in order; the first marker file that exists wins.
var rules = []struct {
	marker string
	stack  Stack
}{
	{"go.mod", Stack{"go", "golang:1.26-alpine", "go test ./..."}},
	{"Cargo.toml", Stack{"rust", "rust:1-alpine", "cargo test"}},
	{"package.json", Stack{"node", "node:20-alpine", "npm test"}},
	{"pyproject.toml", Stack{"python", "python:3.12-alpine", "pytest -q"}},
	{"requirements.txt", Stack{"python", "python:3.12-alpine", "pytest -q"}},
	{"Gemfile", Stack{"ruby", "ruby:3.3-alpine", "bundle exec rake test"}},
}

// Detect returns the first matching stack for dir, or ok=false if none match.
func Detect(dir string) (Stack, bool) {
	for _, r := range rules {
		if _, err := os.Stat(filepath.Join(dir, r.marker)); err == nil {
			return r.stack, true
		}
	}
	return Stack{}, false
}
