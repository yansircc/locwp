package site

import (
	"path/filepath"
	"testing"
)

func newTestConfig(dir string) *Config {
	return &Config{
		Name:    "testsite",
		Port:    10001,
		PHP:     "8.3",
		WPVer:   "latest",
		SiteDir: dir,
		WPRoot:  filepath.Join(dir, "wordpress"),
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	sc := newTestConfig(dir)

	if err := Save(dir, sc); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Name != sc.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, sc.Name)
	}
	if loaded.Port != sc.Port {
		t.Errorf("Port = %d, want %d", loaded.Port, sc.Port)
	}
	if loaded.PHP != sc.PHP {
		t.Errorf("PHP = %q, want %q", loaded.PHP, sc.PHP)
	}
	if loaded.WPRoot != sc.WPRoot {
		t.Errorf("WPRoot = %q, want %q", loaded.WPRoot, sc.WPRoot)
	}
}

func TestLoad_NotExist(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Error("Load() on empty dir should error")
	}
}

func TestStatus_NoCaddyConf(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("LOCWP_HOME", baseDir)

	sc := newTestConfig(baseDir)
	status := Status(sc)
	if status != "stopped" {
		t.Errorf("Status() = %q, want \"stopped\" when no caddy conf exists", status)
	}
}

func TestURL(t *testing.T) {
	sc := &Config{Port: 10005}
	want := "http://localhost:10005"
	if got := sc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}
