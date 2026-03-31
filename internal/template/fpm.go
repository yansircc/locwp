package template

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/site"
)

// HomebrewPrefix returns the Homebrew prefix for the current platform.
func HomebrewPrefix() string {
	if runtime.GOARCH == "arm64" {
		return "/opt/homebrew"
	}
	return "/usr/local"
}

// PHPFormulaName returns the Homebrew formula name for a PHP version.
func PHPFormulaName(version string) string {
	if version == "" {
		version = config.DefaultPHP
	}
	return "php@" + version
}

// FPMPoolDir returns the PHP-FPM pool.d directory for a given PHP version.
func FPMPoolDir(version string) string {
	prefix := HomebrewPrefix()
	return filepath.Join(prefix, "etc", "php", version, "php-fpm.d")
}

// PHPConfDir returns the PHP conf.d directory for a given version.
func PHPConfDir(version string) string {
	return filepath.Join(HomebrewPrefix(), "etc", "php", version, "conf.d")
}

// WritePHPConf writes WordPress-friendly PHP limits to conf.d/locwp.ini.
func WritePHPConf(version string) error {
	dir := PHPConfDir(version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create conf.d dir: %w", err)
	}
	content := `upload_max_filesize = 256M
post_max_size = 256M
memory_limit = 512M
max_execution_time = 300
max_input_vars = 5000
`
	return os.WriteFile(filepath.Join(dir, "locwp.ini"), []byte(content), 0644)
}

func WriteFPMPool(path string, sc *site.Config) error {
	pool := fmt.Sprintf(`[locwp-%d]
user = %s
group = staff
listen = /tmp/locwp-%d.sock
listen.owner = %s
listen.group = staff
listen.mode = 0660

pm = ondemand
pm.max_children = 5
pm.process_idle_timeout = 10s

php_admin_value[error_log] = %s/logs/php-error.log
`, sc.Port, os.Getenv("USER"), sc.Port, os.Getenv("USER"), sc.SiteDir)

	return os.WriteFile(path, []byte(pool), 0644)
}
