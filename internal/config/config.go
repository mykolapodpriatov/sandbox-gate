// Package config loads an optional .sandbox-gate.yml from the project root. Every
// field is optional; pointers distinguish "unset" (fall through to a flag or
// default) from an explicit false/zero.
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// File mirrors .sandbox-gate.yml.
type File struct {
	Image     string   `yaml:"image"`
	Cmd       string   `yaml:"cmd"`
	Backend   string   `yaml:"backend"`
	Network   *bool    `yaml:"network"`
	MemoryMB  int      `yaml:"memory_mb"`
	CPUs      float64  `yaml:"cpus"`
	PidsLimit int      `yaml:"pids_limit"`
	NonRoot   *bool    `yaml:"non_root"`
	Timeout   string   `yaml:"timeout"`
}

// Load reads path. A missing file is not an error — it returns the zero File.
func Load(path string) (File, error) {
	if path == "" {
		path = ".sandbox-gate.yml"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

// ParseTimeout parses the Timeout string (e.g. "120s", "2m"); "" → 0.
func (f File) ParseTimeout() (time.Duration, error) {
	if f.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(f.Timeout)
}
