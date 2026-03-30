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
	SiteDir    string `json:"site_dir"`
	WPRoot     string `json:"wp_root"`
	AdminUser  string `json:"admin_user"`
	AdminPass  string `json:"admin_pass"`
	AdminEmail string `json:"admin_email"`
}

// URL returns the HTTP URL for the site.
func (sc *Config) URL() string {
	return fmt.Sprintf("http://localhost:%d", sc.Port)
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

// CaddyConfPath returns the path to the Caddy site config for a site.
func CaddyConfPath(name string) string {
	return filepath.Join(config.CaddySitesDir(), name+".caddy")
}

// CaddyConfEnabled reports whether the site's Caddy config is active.
func CaddyConfEnabled(name string) bool {
	_, err := os.Stat(CaddyConfPath(name))
	return err == nil
}

// Status checks if a site is responding.
func Status(sc *Config) string {
	if !CaddyConfEnabled(sc.Name) {
		return "stopped"
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", sc.Port), 500*time.Millisecond)
	if err != nil {
		return "stopped"
	}
	conn.Close()
	return "running"
}
