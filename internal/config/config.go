package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

const (
	dirName      = ".locwp"
	portStart    = 8081
	portEnd      = 8180
)

// BaseDir returns the locwp data directory, creating it if needed.
// Honors LOCWP_HOME env var, defaults to ~/.locwp.
func BaseDir() string {
	dir := os.Getenv("LOCWP_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dirName)
	}
	os.MkdirAll(dir, 0755)
	return dir
}

// usedPorts reads all existing site configs and returns a set of assigned ports.
func usedPorts(baseDir string) map[int]bool {
	used := make(map[int]bool)
	sitesDir := filepath.Join(baseDir, "sites")
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		return used
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sitesDir, e.Name(), "config.json"))
		if err != nil {
			continue
		}
		var cfg struct {
			Port int `json:"port"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.Port > 0 {
			used[cfg.Port] = true
		}
	}
	return used
}

// NextPort finds the next available port in the range.
// A port is available only if it is not assigned to any existing site
// AND not currently bound by another process.
func NextPort(baseDir string) (int, error) {
	used := usedPorts(baseDir)
	for p := portStart; p <= portEnd; p++ {
		if used[p] {
			continue
		}
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			ln.Close()
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", portStart, portEnd)
}
