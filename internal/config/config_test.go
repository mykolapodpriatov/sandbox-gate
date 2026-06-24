package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFileIsZero(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "absent.yml"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if f.Image != "" || f.Network != nil {
		t.Errorf("want zero File, got %+v", f)
	}
}

func TestLoad_Parse(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yml")
	os.WriteFile(p, []byte("image: node:20\ncmd: npm test\nnetwork: true\nmemory_mb: 256\ntimeout: 90s\n"), 0o644)

	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.Image != "node:20" || f.Cmd != "npm test" || f.MemoryMB != 256 {
		t.Errorf("scalars: %+v", f)
	}
	if f.Network == nil || !*f.Network {
		t.Error("network pointer should be explicit true")
	}
	d, err := f.ParseTimeout()
	if err != nil || d.String() != "1m30s" {
		t.Errorf("timeout: %v (%v)", d, err)
	}
}

func TestParseTimeout_Empty(t *testing.T) {
	d, err := File{}.ParseTimeout()
	if err != nil || d != 0 {
		t.Errorf("empty timeout: %v %v", d, err)
	}
}
