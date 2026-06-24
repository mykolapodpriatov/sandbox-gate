package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_KnownStacks(t *testing.T) {
	for _, c := range []struct{ marker, name string }{
		{"go.mod", "go"},
		{"package.json", "node"},
		{"requirements.txt", "python"},
		{"Cargo.toml", "rust"},
		{"Gemfile", "ruby"},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, c.marker), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		st, ok := Detect(dir)
		if !ok || st.Name != c.name {
			t.Errorf("%s: got name=%q ok=%v, want %q", c.marker, st.Name, ok, c.name)
		}
		if st.Image == "" || st.Cmd == "" {
			t.Errorf("%s: incomplete stack %+v", c.marker, st)
		}
	}
}

func TestDetect_NoneInEmptyDir(t *testing.T) {
	if _, ok := Detect(t.TempDir()); ok {
		t.Error("empty dir must not detect a stack")
	}
}

func TestDetect_OrderGoBeatsNode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("x"), 0o644)
	if st, _ := Detect(dir); st.Name != "go" {
		t.Errorf("first rule should win: got %q", st.Name)
	}
}
