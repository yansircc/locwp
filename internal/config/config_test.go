package config

import (
	"encoding/json"
	"fmt"
	"net"
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

// TestNextPort_SkipsConfiguredPorts verifies that NextPort skips ports
// already assigned in existing site config.json files (the core fix).
func TestNextPort_SkipsConfiguredPorts(t *testing.T) {
	tmp := t.TempDir()
	sitesDir := filepath.Join(tmp, "sites")

	// Pre-create 3 sites all claiming port 8081 (simulates the old bug)
	for i := 1; i <= 3; i++ {
		dir := filepath.Join(sitesDir, fmt.Sprintf("site-%d", i))
		os.MkdirAll(dir, 0755)
		data, _ := json.Marshal(map[string]int{"port": portStart})
		os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
	}

	port, err := NextPort(tmp)
	if err != nil {
		t.Fatalf("NextPort() error: %v", err)
	}
	if port == portStart {
		t.Errorf("NextPort() = %d, should have skipped it (already configured)", portStart)
	}
	if port != portStart+1 {
		t.Errorf("NextPort() = %d, want %d", port, portStart+1)
	}
}

// TestNextPort_SkipsMultipleConfiguredPorts verifies consecutive ports are skipped.
func TestNextPort_SkipsMultipleConfiguredPorts(t *testing.T) {
	tmp := t.TempDir()
	sitesDir := filepath.Join(tmp, "sites")

	// Occupy the first 5 ports
	for i := 0; i < 5; i++ {
		dir := filepath.Join(sitesDir, fmt.Sprintf("site-%d", i))
		os.MkdirAll(dir, 0755)
		data, _ := json.Marshal(map[string]int{"port": portStart + i})
		os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
	}

	port, err := NextPort(tmp)
	if err != nil {
		t.Fatalf("NextPort() error: %v", err)
	}
	if port != portStart+5 {
		t.Errorf("NextPort() = %d, want %d (first 5 ports occupied)", port, portStart+5)
	}
}

// TestNextPort_SkipsBoundPort verifies that a port held by another process is skipped.
func TestNextPort_SkipsBoundPort(t *testing.T) {
	tmp := t.TempDir()

	// Bind portStart so net.Listen will fail for it
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portStart))
	if err != nil {
		t.Skipf("cannot bind port %d for test: %v", portStart, err)
	}
	defer ln.Close()

	port, err := NextPort(tmp)
	if err != nil {
		t.Fatalf("NextPort() error: %v", err)
	}
	if port == portStart {
		t.Errorf("NextPort() = %d, should have skipped it (port is bound)", portStart)
	}
}

// TestUsedPorts verifies the helper correctly reads assigned ports.
func TestUsedPorts(t *testing.T) {
	tmp := t.TempDir()
	sitesDir := filepath.Join(tmp, "sites")

	// No sites dir → empty map
	got := usedPorts(tmp)
	if len(got) != 0 {
		t.Errorf("usedPorts with no sites dir: got %d entries, want 0", len(got))
	}

	// Create 2 sites with distinct ports
	for _, p := range []int{8081, 8085} {
		dir := filepath.Join(sitesDir, fmt.Sprintf("site-%d", p))
		os.MkdirAll(dir, 0755)
		data, _ := json.Marshal(map[string]int{"port": p})
		os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
	}

	got = usedPorts(tmp)
	if len(got) != 2 {
		t.Fatalf("usedPorts: got %d entries, want 2", len(got))
	}
	if !got[8081] || !got[8085] {
		t.Errorf("usedPorts: expected {8081: true, 8085: true}, got %v", got)
	}
}
