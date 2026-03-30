package site

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/yansircc/locwp/internal/config"
)

type Config struct {
	Port       int    `json:"port"`
	PHP        string `json:"php"`
	WPVer      string `json:"wp_version"`
	SiteDir    string `json:"site_dir"`
	WPRoot     string `json:"wp_root"`
	AdminUser  string `json:"admin_user"`
	AdminPass  string `json:"admin_pass"`
	AdminEmail string `json:"admin_email"`
}

// PortStr returns the port as a string.
func (sc *Config) PortStr() string {
	return strconv.Itoa(sc.Port)
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

// LoadByPort finds and loads a site by port number.
func LoadByPort(port int) (*Config, error) {
	portStr := strconv.Itoa(port)
	siteDir := filepath.Join(config.BaseDir(), "sites", portStr)
	sc, err := Load(siteDir)
	if err != nil {
		return nil, fmt.Errorf("site %s not found: %w", portStr, err)
	}
	return sc, nil
}

// CaddyConfPath returns the path to the Caddy site config.
func CaddyConfPath(port int) string {
	return filepath.Join(config.CaddySitesDir(), strconv.Itoa(port)+".caddy")
}

// CaddyConfEnabled reports whether the site's Caddy config is active.
func CaddyConfEnabled(port int) bool {
	_, err := os.Stat(CaddyConfPath(port))
	return err == nil
}

// Status checks if a site is responding.
func Status(sc *Config) string {
	if !CaddyConfEnabled(sc.Port) {
		return "stopped"
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", sc.Port), 500*time.Millisecond)
	if err != nil {
		return "stopped"
	}
	conn.Close()
	return "running"
}
