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
// For PHP 8.3 (the default Homebrew PHP), it returns "php".
// For other versions, it returns "php@X.Y".
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

func WriteFPMPool(path string, sc *site.Config) error {
	pool := fmt.Sprintf(`[%s]
user = %s
group = staff
listen = /tmp/locwp-%s.sock
listen.owner = %s
listen.group = staff
listen.mode = 0660

pm = ondemand
pm.max_children = 5
pm.process_idle_timeout = 10s

php_admin_value[error_log] = %s/logs/php-error.log
`, sc.Name, os.Getenv("USER"), sc.Name, os.Getenv("USER"), sc.SiteDir)

	return os.WriteFile(path, []byte(pool), 0644)
}
