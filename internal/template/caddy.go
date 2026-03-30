package template

import (
	"fmt"
	"os"

	"github.com/yansircc/locwp/internal/site"
)

// WriteCaddyConf writes a Caddy site config block to the given path.
func WriteCaddyConf(path string, sc *site.Config) error {
	conf := fmt.Sprintf(`:%d {
	root * %s
	php_fastcgi unix//tmp/locwp-%s.sock
	file_server
	encode gzip

	log {
		output file %s/logs/access.log
	}
}
`, sc.Port, sc.WPRoot, sc.Name, sc.SiteDir)

	return os.WriteFile(path, []byte(conf), 0644)
}
