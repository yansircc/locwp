package site

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestConfig(dir string) *Config {
	return &Config{
		Name:    "testsite",
		Port:    8099,
		PHP:     "8.3",
		WPVer:   "latest",
		DBName:  "wp_testsite",
		DBUser:  "root",
		DBHost:  "127.0.0.1",
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
	if loaded.DBName != sc.DBName {
		t.Errorf("DBName = %q, want %q", loaded.DBName, sc.DBName)
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

func TestVhostEnableDisable(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("LOCWP_HOME", baseDir)

	name := "mysite"
	nginxDir := filepath.Join(baseDir, "nginx", "sites")
	os.MkdirAll(nginxDir, 0755)

	// Create an enabled vhost
	confPath := filepath.Join(nginxDir, name+".conf")
	os.WriteFile(confPath, []byte("server {}"), 0644)

	// Should be enabled
	if !VhostEnabled(name) {
		t.Error("VhostEnabled() = false after creating .conf")
	}

	// Disable it
	if err := DisableVhost(name); err != nil {
		t.Fatalf("DisableVhost() error: %v", err)
	}
	if VhostEnabled(name) {
		t.Error("VhostEnabled() = true after DisableVhost()")
	}

	// .conf.disabled should exist
	if _, err := os.Stat(filepath.Join(nginxDir, name+".conf.disabled")); err != nil {
		t.Error(".conf.disabled file not found after DisableVhost()")
	}

	// Disable again should be idempotent
	if err := DisableVhost(name); err != nil {
		t.Fatalf("DisableVhost() second call error: %v", err)
	}

	// Re-enable
	if err := EnableVhost(name); err != nil {
		t.Fatalf("EnableVhost() error: %v", err)
	}
	if !VhostEnabled(name) {
		t.Error("VhostEnabled() = false after EnableVhost()")
	}

	// Enable again should be idempotent
	if err := EnableVhost(name); err != nil {
		t.Fatalf("EnableVhost() second call error: %v", err)
	}
}

func TestVhostEnable_NoFile(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("LOCWP_HOME", baseDir)

	nginxDir := filepath.Join(baseDir, "nginx", "sites")
	os.MkdirAll(nginxDir, 0755)

	err := EnableVhost("nonexistent")
	if err == nil {
		t.Error("EnableVhost() should error when no conf file exists")
	}
}

func TestStatus_NoVhost(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("LOCWP_HOME", baseDir)

	sc := newTestConfig(baseDir)
	status := Status(sc)
	if status != "stopped" {
		t.Errorf("Status() = %q, want \"stopped\" when no vhost exists", status)
	}
}
