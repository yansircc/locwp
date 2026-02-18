package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yansircc/locwp/internal/site"
)

func testSiteConfig(dir string) *site.Config {
	return &site.Config{
		Name:    "demo",
		Port:    8081,
		PHP:     "8.2",
		WPVer:   "6.4",
		DBName:  "wp_demo",
		DBUser:  "root",
		DBHost:  "127.0.0.1",
		SiteDir: dir,
		WPRoot:  filepath.Join(dir, "wordpress"),
	}
}

func TestPHPFormulaName(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"8.3", "php@8.3"},
		{"8.2", "php@8.2"},
		{"8.1", "php@8.1"},
		{"", "php@8.3"}, // default
	}
	for _, tt := range tests {
		got := PHPFormulaName(tt.version)
		if got != tt.want {
			t.Errorf("PHPFormulaName(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

func TestHomebrewPrefix(t *testing.T) {
	prefix := HomebrewPrefix()
	if runtime.GOARCH == "arm64" {
		if prefix != "/opt/homebrew" {
			t.Errorf("HomebrewPrefix() = %q on arm64, want /opt/homebrew", prefix)
		}
	} else {
		if prefix != "/usr/local" {
			t.Errorf("HomebrewPrefix() = %q on non-arm64, want /usr/local", prefix)
		}
	}
}

func TestFPMPoolDir(t *testing.T) {
	dir := FPMPoolDir("8.2")
	prefix := HomebrewPrefix()
	want := filepath.Join(prefix, "etc", "php", "8.2", "php-fpm.d")
	if dir != want {
		t.Errorf("FPMPoolDir(\"8.2\") = %q, want %q", dir, want)
	}
}

func TestWriteNginxConf(t *testing.T) {
	dir := t.TempDir()
	sc := testSiteConfig(dir)
	outPath := filepath.Join(dir, "test.conf")

	if err := WriteNginxConf(outPath, sc); err != nil {
		t.Fatalf("WriteNginxConf() error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)

	checks := []string{
		"listen 8081",
		"root " + sc.WPRoot,
		"fastcgi_pass unix:/tmp/locwp-demo.sock",
		"access_log " + dir + "/logs/access.log",
		"try_files",
		"index.php",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("nginx conf missing %q", check)
		}
	}
}

func TestWriteFPMPool(t *testing.T) {
	dir := t.TempDir()
	sc := testSiteConfig(dir)
	outPath := filepath.Join(dir, "test-fpm.conf")

	if err := WriteFPMPool(outPath, sc); err != nil {
		t.Fatalf("WriteFPMPool() error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)

	checks := []string{
		"[demo]",
		"listen = /tmp/locwp-demo.sock",
		"pm = ondemand",
		"pm.max_children = 5",
		"php_admin_value[error_log]",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("FPM pool missing %q", check)
		}
	}
}

func TestWritePawlConfig(t *testing.T) {
	dir := t.TempDir()
	sc := testSiteConfig(dir)
	outPath := filepath.Join(dir, "config.json")

	if err := WritePawlConfig(outPath, sc); err != nil {
		t.Fatalf("WritePawlConfig() error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	// Should be valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("pawl config is not valid JSON: %v", err)
	}

	// Check vars
	vars, ok := raw["vars"].(map[string]interface{})
	if !ok {
		t.Fatal("pawl config missing 'vars'")
	}
	if vars["site"] != "demo" {
		t.Errorf("vars.site = %q, want \"demo\"", vars["site"])
	}
	if vars["port"] != "8081" {
		t.Errorf("vars.port = %q, want \"8081\"", vars["port"])
	}
	if vars["db_name"] != "wp_demo" {
		t.Errorf("vars.db_name = %q, want \"wp_demo\"", vars["db_name"])
	}

	// Check workflow steps
	workflow, ok := raw["workflow"].([]interface{})
	if !ok {
		t.Fatal("pawl config missing 'workflow'")
	}

	expectedSteps := []string{
		"check-deps", "create-db", "download-wp",
		"gen-wp-config", "reload-services", "install-wp", "verify",
	}
	if len(workflow) != len(expectedSteps) {
		t.Fatalf("workflow has %d steps, want %d", len(workflow), len(expectedSteps))
	}
	for i, step := range workflow {
		s := step.(map[string]interface{})
		if s["name"] != expectedSteps[i] {
			t.Errorf("workflow[%d].name = %q, want %q", i, s["name"], expectedSteps[i])
		}
	}

	// Verify step should have "verify": "manual"
	lastStep := workflow[len(workflow)-1].(map[string]interface{})
	if lastStep["verify"] != "manual" {
		t.Error("last step should have verify=manual")
	}
}
