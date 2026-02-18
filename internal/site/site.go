package site

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/yansircc/locwp/internal/config"
)

type Config struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	PHP        string `json:"php"`
	WPVer      string `json:"wp_version"`
	DBName     string `json:"db_name"`
	DBUser     string `json:"db_user"`
	DBHost     string `json:"db_host"`
	SiteDir    string `json:"site_dir"`
	WPRoot     string `json:"wp_root"`
	AdminUser  string `json:"admin_user"`
	AdminPass  string `json:"admin_pass"`
	AdminEmail string `json:"admin_email"`
}

// Save writes site config to site_dir/config.json.
func Save(siteDir string, sc *Config) error {
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(siteDir, "config.json"), data, 0644)
}

// Load reads site config from a site directory.
func Load(siteDir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(siteDir, "config.json"))
	if err != nil {
		return nil, err
	}
	var sc Config
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

// LoadByName finds and loads a site by name.
func LoadByName(name string) (*Config, error) {
	siteDir := filepath.Join(config.BaseDir(), "sites", name)
	sc, err := Load(siteDir)
	if err != nil {
		return nil, fmt.Errorf("site %q not found: %w", name, err)
	}
	return sc, nil
}

// VhostPath returns the path to the nginx vhost conf for a site.
func VhostPath(name string) string {
	return filepath.Join(config.BaseDir(), "nginx", "sites", name+".conf")
}

// VhostDisabledPath returns the path to a disabled nginx vhost conf.
func VhostDisabledPath(name string) string {
	return filepath.Join(config.BaseDir(), "nginx", "sites", name+".conf.disabled")
}

// VhostEnabled reports whether the site's nginx vhost conf is enabled.
func VhostEnabled(name string) bool {
	_, err := os.Stat(VhostPath(name))
	return err == nil
}

// EnableVhost renames <name>.conf.disabled back to <name>.conf.
func EnableVhost(name string) error {
	enabled := VhostPath(name)
	disabled := VhostDisabledPath(name)

	if _, err := os.Stat(enabled); err == nil {
		return nil // already enabled
	}
	if _, err := os.Stat(disabled); err != nil {
		return fmt.Errorf("vhost config not found for site %q", name)
	}
	return os.Rename(disabled, enabled)
}

// DisableVhost renames <name>.conf to <name>.conf.disabled.
func DisableVhost(name string) error {
	enabled := VhostPath(name)
	disabled := VhostDisabledPath(name)

	if _, err := os.Stat(disabled); err == nil {
		return nil // already disabled
	}
	if _, err := os.Stat(enabled); err != nil {
		return fmt.Errorf("vhost config not found for site %q", name)
	}
	return os.Rename(enabled, disabled)
}

// Status checks if a site is responding.
func Status(sc *Config) string {
	if !VhostEnabled(sc.Name) {
		return "stopped"
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", sc.Port), 500*time.Millisecond)
	if err != nil {
		return "stopped"
	}
	conn.Close()
	return "running"
}
