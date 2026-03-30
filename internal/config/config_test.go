package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBaseDir_Default(t *testing.T) {
	os.Unsetenv("LOCWP_HOME")
	dir := BaseDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".locwp")
	if dir != want {
		t.Errorf("BaseDir() = %q, want %q", dir, want)
	}
}

func TestBaseDir_EnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCWP_HOME", tmp)
	dir := BaseDir()
	if dir != tmp {
		t.Errorf("BaseDir() = %q, want %q", dir, tmp)
	}
}

func TestNextPort_NoSites(t *testing.T) {
	tmp := t.TempDir()
	port := NextPort(tmp)
	if port != StartPort {
		t.Errorf("NextPort() = %d, want %d", port, StartPort)
	}
}

func TestNextPort_WithExisting(t *testing.T) {
	tmp := t.TempDir()
	siteDir := filepath.Join(tmp, "sites", "mysite")
	os.MkdirAll(siteDir, 0755)
	data, _ := json.Marshal(map[string]interface{}{"port": 10003})
	os.WriteFile(filepath.Join(siteDir, "config.json"), data, 0644)

	port := NextPort(tmp)
	if port != 10004 {
		t.Errorf("NextPort() = %d, want 10004", port)
	}
}

func TestNextPort_MultipleSites(t *testing.T) {
	tmp := t.TempDir()
	for _, s := range []struct {
		name string
		port int
	}{
		{"site1", 10001},
		{"site2", 10005},
		{"site3", 10003},
	} {
		dir := filepath.Join(tmp, "sites", s.name)
		os.MkdirAll(dir, 0755)
		data, _ := json.Marshal(map[string]interface{}{"port": s.port})
		os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
	}

	port := NextPort(tmp)
	if port != 10006 {
		t.Errorf("NextPort() = %d, want 10006", port)
	}
}

func TestCaddySitesDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCWP_HOME", tmp)
	dir := CaddySitesDir()
	want := filepath.Join(tmp, "caddy", "sites")
	if dir != want {
		t.Errorf("CaddySitesDir() = %q, want %q", dir, want)
	}
}
