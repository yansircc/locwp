package config

import (
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

func TestNextPort(t *testing.T) {
	tmp := t.TempDir()
	port, err := NextPort(tmp)
	if err != nil {
		t.Fatalf("NextPort() error: %v", err)
	}
	if port < portStart || port > portEnd {
		t.Errorf("NextPort() = %d, want in range [%d, %d]", port, portStart, portEnd)
	}
}
