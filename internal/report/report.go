// Package report renders a run Result as machine-readable JSON and a one-line
// human summary.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mykolapodpriatov/sandbox-gate/internal/sandbox"
)

// WriteJSON writes the result as pretty JSON to path. "" or "-" is a no-op
// (the caller may still print the summary).
func WriteJSON(path string, r sandbox.Result) error {
	if path == "" || path == "-" {
		return nil
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// Summary prints a single human-readable status line.
func Summary(w io.Writer, r sandbox.Result) {
	icon := "✗"
	if r.Passed() {
		icon = "✓"
	}
	net := "off"
	if r.Isolation.Network {
		net = "ON"
	}
	fmt.Fprintf(w, "\n%s sandbox-gate: %s  (exit %d · %dms · backend=%s · image=%s · net=%s)\n",
		icon, r.Status, r.ExitCode, r.DurationMS, r.Backend, r.Image, net)
	if r.Err != "" {
		fmt.Fprintf(w, "  ! %s\n", r.Err)
	}
}
