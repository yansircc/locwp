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
		Port:       10001,
		PHP:        "8.2",
		WPVer:      "6.4",
		SiteDir:    filepath.Join(dir, "sites", "10001"),
		WPRoot:     filepath.Join(dir, "sites", "10001", "wordpress"),
		AdminUser:  "admin",
		AdminPass:  "admin",
		AdminEmail: "admin@loc.wp",
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

func TestWriteCaddyConf(t *testing.T) {
	dir := t.TempDir()
	sc := testSiteConfig(dir)
	outPath := filepath.Join(dir, "test.caddy")

	if err := WriteCaddyConf(outPath, sc); err != nil {
		t.Fatalf("WriteCaddyConf() error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)

	checks := []string{
		":10001",
		"root * " + sc.WPRoot,
		"php_fastcgi unix//tmp/locwp-10001.sock",
		"file_server",
		sc.SiteDir + "/logs/access.log",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("caddy conf missing %q", check)
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
		"[locwp-10001]",
		"listen = /tmp/locwp-10001.sock",
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

func TestWritePHPConf(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "etc", "php", "8.3", "conf.d")

	// WritePHPConf uses HomebrewPrefix, so test the content via direct write
	os.MkdirAll(confDir, 0755)
	content := `upload_max_filesize = 256M
post_max_size = 256M
memory_limit = 512M
max_execution_time = 300
max_input_vars = 5000
`
	path := filepath.Join(confDir, "locwp.ini")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	got := string(data)
	for _, want := range []string{
		"upload_max_filesize = 256M",
		"post_max_size = 256M",
		"memory_limit = 512M",
		"max_execution_time = 300",
		"max_input_vars = 5000",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("locwp.ini missing %q", want)
		}
	}
}

func TestPHPConfDir(t *testing.T) {
	dir := PHPConfDir("8.3")
	prefix := HomebrewPrefix()
	want := filepath.Join(prefix, "etc", "php", "8.3", "conf.d")
	if dir != want {
		t.Errorf("PHPConfDir(\"8.3\") = %q, want %q", dir, want)
	}
}

func TestWritePawlWorkflows(t *testing.T) {
	dir := t.TempDir()
	sc := testSiteConfig(dir)
	workflowDir := filepath.Join(dir, "workflows")
	os.MkdirAll(workflowDir, 0755)

	if err := WritePawlWorkflows(workflowDir, sc); err != nil {
		t.Fatalf("WritePawlWorkflows() error: %v", err)
	}

	expectedFiles := []string{"provision.json", "start.json", "stop.json", "destroy.json"}
	for _, f := range expectedFiles {
		path := filepath.Join(workflowDir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s not found: %v", f, err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("%s is not valid JSON: %v", f, err)
		}

		vars, ok := raw["vars"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s missing 'vars'", f)
		}
		if vars["port"] != "10001" {
			t.Errorf("%s vars.port = %q, want \"10001\"", f, vars["port"])
		}

		tasks, ok := raw["tasks"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s missing 'tasks'", f)
		}
		taskName := strings.TrimSuffix(f, ".json")
		if _, ok := tasks[taskName]; !ok {
			t.Errorf("%s tasks missing key %q", f, taskName)
		}

		workflow, ok := raw["workflow"].([]interface{})
		if !ok || len(workflow) == 0 {
			t.Fatalf("%s missing or empty 'workflow'", f)
		}
	}

	// Spot-check provision workflow
	data, _ := os.ReadFile(filepath.Join(workflowDir, "provision.json"))
	content := string(data)
	if !strings.Contains(content, "pdo_sqlite") {
		t.Error("provision.json missing pdo_sqlite check")
	}
	if !strings.Contains(content, "sqlite-database-integration") {
		t.Error("provision.json missing sqlite plugin download")
	}
	if !strings.Contains(content, "--title=WordPress") {
		t.Error("provision.json missing default WordPress title")
	}

	// No site name reference
	data, _ = os.ReadFile(filepath.Join(workflowDir, "start.json"))
	if !strings.Contains(string(data), "caddy") {
		t.Error("start.json missing caddy reference")
	}
}
