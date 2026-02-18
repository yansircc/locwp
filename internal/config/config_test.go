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

func TestDomainExists_NoSites(t *testing.T) {
	tmp := t.TempDir()
	if DomainExists(tmp, "foo.loc.wp") {
		t.Error("DomainExists should return false when no sites dir exists")
	}
}

func TestDomainExists_Match(t *testing.T) {
	tmp := t.TempDir()
	siteDir := filepath.Join(tmp, "sites", "mysite")
	os.MkdirAll(siteDir, 0755)
	data, _ := json.Marshal(map[string]string{"domain": "mysite.loc.wp"})
	os.WriteFile(filepath.Join(siteDir, "config.json"), data, 0644)

	if !DomainExists(tmp, "mysite.loc.wp") {
		t.Error("DomainExists should return true for existing domain")
	}
}

func TestDomainExists_NoMatch(t *testing.T) {
	tmp := t.TempDir()
	siteDir := filepath.Join(tmp, "sites", "mysite")
	os.MkdirAll(siteDir, 0755)
	data, _ := json.Marshal(map[string]string{"domain": "mysite.loc.wp"})
	os.WriteFile(filepath.Join(siteDir, "config.json"), data, 0644)

	if DomainExists(tmp, "other.loc.wp") {
		t.Error("DomainExists should return false for non-matching domain")
	}
}
