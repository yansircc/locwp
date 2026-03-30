package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const dirName = ".locwp"

// DefaultPHP is the default PHP version used across setup and add commands.
const DefaultPHP = "8.3"

// StartPort is the first port allocated to sites.
const StartPort = 10001

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

// CaddySitesDir returns the path to per-site Caddy config directory.
func CaddySitesDir() string {
	return filepath.Join(BaseDir(), "caddy", "sites")
}

// NextPort scans all site configs and returns the next available port.
func NextPort(baseDir string) int {
	sitesDir := filepath.Join(baseDir, "sites")
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		return StartPort
	}
	maxPort := StartPort - 1
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
		if json.Unmarshal(data, &cfg) == nil && cfg.Port > maxPort {
			maxPort = cfg.Port
		}
	}
	return maxPort + 1
}
