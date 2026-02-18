package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const dirName = ".locwp"

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

// SSLDir returns the path to the SSL certificate directory (~/.locwp/ssl/).
func SSLDir() string {
	return filepath.Join(BaseDir(), "ssl")
}

// DomainExists checks whether any existing site already uses the given domain.
func DomainExists(baseDir, domain string) bool {
	sitesDir := filepath.Join(baseDir, "sites")
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		return false
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
			Domain string `json:"domain"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.Domain == domain {
			return true
		}
	}
	return false
}
