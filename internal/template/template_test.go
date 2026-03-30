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
		Name:       "demo",
		Port:       10001,
		PHP:        "8.2",
		WPVer:      "6.4",
		SiteDir:    filepath.Join(dir, "sites", "demo"),
		WPRoot:     filepath.Join(dir, "sites", "demo", "wordpress"),
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
		"php_fastcgi unix//tmp/locwp-demo.sock",
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

		// All workflows must have vars, tasks, and workflow
		vars, ok := raw["vars"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s missing 'vars'", f)
		}
		if vars["site"] != "demo" {
			t.Errorf("%s vars.site = %q, want \"demo\"", f, vars["site"])
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

	// Spot-check provision workflow has SQLite-related steps
	data, _ := os.ReadFile(filepath.Join(workflowDir, "provision.json"))
	content := string(data)
	if !strings.Contains(content, "pdo_sqlite") {
		t.Error("provision.json missing pdo_sqlite check")
	}
	if !strings.Contains(content, "sqlite-database-integration") {
		t.Error("provision.json missing sqlite plugin download")
	}
	if !strings.Contains(content, "db.copy") {
		t.Error("provision.json missing db.php drop-in setup")
	}
	if !strings.Contains(content, "install-wp") {
		t.Error("provision.json missing install-wp step")
	}
	if !strings.Contains(content, "set-permalinks") {
		t.Error("provision.json missing set-permalinks step")
	}

	// Spot-check destroy workflow has NO mariadb references
	data, _ = os.ReadFile(filepath.Join(workflowDir, "destroy.json"))
	if strings.Contains(string(data), "mariadb") {
		t.Error("destroy.json should not reference mariadb")
	}

	// Check that start.json references caddy (not nginx)
	data, _ = os.ReadFile(filepath.Join(workflowDir, "start.json"))
	if !strings.Contains(string(data), "caddy") {
		t.Error("start.json missing caddy reference")
	}
	if strings.Contains(string(data), "nginx") {
		t.Error("start.json should not reference nginx")
	}
}
