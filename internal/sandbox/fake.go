package sandbox

import (
	"context"
	"io"
)

// Fake is a deterministic, in-memory backend used by tests. It records the Spec
// it was given and returns a preset Result, so the gate orchestration can be
// tested without a Docker daemon.
type Fake struct {
	Out    string // streamed to the writer
	Result Result
	Err    error

	Got    Spec // captured for assertions
	Called int
}

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Run(_ context.Context, spec Spec, out io.Writer) (Result, error) {
	f.Got = spec
	f.Called++
	if f.Out != "" {
		_, _ = io.WriteString(out, f.Out)
	}
	r := f.Result
	r.Backend = "fake"
	r.Image = spec.Image
	r.Isolation = spec.Isolation
	return r, f.Err
}
