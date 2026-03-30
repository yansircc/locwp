package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/site"
)

type pawlConfig struct {
	Vars     map[string]string       `json:"vars"`
	Tasks    map[string]pawlTaskDecl `json:"tasks"`
	Workflow []pawlStep              `json:"workflow"`
}

type pawlTaskDecl struct {
	Description string `json:"description,omitempty"`
}

type pawlStep struct {
	Name   string `json:"name"`
	Run    string `json:"run,omitempty"`
	OnFail string `json:"on_fail,omitempty"`
}

const sqlitePluginURL = "https://downloads.wordpress.org/plugin/sqlite-database-integration.latest-stable.zip"

// WritePawlWorkflows generates all lifecycle workflow files under workflowDir.
func WritePawlWorkflows(workflowDir string, sc *site.Config) error {
	prefix := HomebrewPrefix()
	phpBin := filepath.Join(prefix, "opt", "php@"+sc.PHP, "bin", "php")
	baseDir := filepath.Dir(filepath.Dir(sc.SiteDir))

	vars := map[string]string{
		"site":             sc.Name,
		"port":             fmt.Sprintf("%d", sc.Port),
		"wp_root":          sc.WPRoot,
		"wp_ver":           sc.WPVer,
		"php_ver":          sc.PHP,
		"php_bin":          phpBin,
		"site_dir":         sc.SiteDir,
		"admin_user":       sc.AdminUser,
		"admin_pass":       sc.AdminPass,
		"admin_email":      sc.AdminEmail,
		"caddy_conf":       filepath.Join(config.CaddySitesDir(), sc.Name+".caddy"),
		"fpm_local":        filepath.Join(baseDir, "php", sc.Name+".conf"),
		"fpm_pool":         filepath.Join(FPMPoolDir(sc.PHP), "locwp-"+sc.Name+".conf"),
		"sqlite_plugin_url": sqlitePluginURL,
	}

	type workflowDef struct {
		description string
		steps       []pawlStep
	}

	workflows := map[string]workflowDef{
		"provision": {
			description: "Provision WordPress site",
			steps: []pawlStep{
				{Name: "check-deps", Run: "test -x ${php_bin} && which caddy wp && ${php_bin} -m | grep -q pdo_sqlite"},
				{Name: "download-wp", Run: "${php_bin} -d memory_limit=512M $(which wp) core download --path=${wp_root} --version=${wp_ver}", OnFail: "retry"},
				{Name: "download-sqlite-plugin", Run: "curl -sL ${sqlite_plugin_url} -o /tmp/locwp-sqlite-plugin.zip && unzip -qo /tmp/locwp-sqlite-plugin.zip -d ${wp_root}/wp-content/plugins/ && rm -f /tmp/locwp-sqlite-plugin.zip", OnFail: "retry"},
				{Name: "setup-db-dropin", Run: "cp ${wp_root}/wp-content/plugins/sqlite-database-integration/db.copy ${wp_root}/wp-content/db.php && mkdir -p ${wp_root}/wp-content/database"},
				{Name: "gen-wp-config", Run: "${php_bin} -d memory_limit=512M $(which wp) config create --path=${wp_root} --dbname=wordpress --dbuser=unused --dbhost=unused --skip-check"},
				{Name: "configure-sqlite", Run: "${php_bin} -d memory_limit=512M $(which wp) config set DB_DIR ${wp_root}/wp-content/database --path=${wp_root} --type=constant && ${php_bin} -d memory_limit=512M $(which wp) config set DB_FILE .ht.sqlite --path=${wp_root} --type=constant"},
				{Name: "provision-services", Run: "brew services restart php@${php_ver} 2>/dev/null; brew services restart caddy", OnFail: "retry"},
				{Name: "install-wp", Run: "${php_bin} -d memory_limit=512M $(which wp) core install --path=${wp_root} --url=http://localhost:${port} --title=${site} --admin_user=${admin_user} --admin_password=${admin_pass} --admin_email=${admin_email}", OnFail: "retry"},
				{Name: "set-permalinks", Run: "${php_bin} -d memory_limit=512M $(which wp) rewrite structure '/%postname%/' --path=${wp_root} && ${php_bin} -d memory_limit=512M $(which wp) rewrite flush --path=${wp_root}"},
			},
		},
		"start": {
			description: "Start WordPress site",
			steps: []pawlStep{
				{Name: "enable-caddy-conf", Run: "mv ${caddy_conf}.disabled ${caddy_conf} 2>/dev/null || true"},
				{Name: "start-php", Run: "brew services start php@${php_ver}"},
				{Name: "reload-caddy", Run: "brew services restart caddy"},
			},
		},
		"stop": {
			description: "Stop WordPress site",
			steps: []pawlStep{
				{Name: "disable-caddy-conf", Run: "mv ${caddy_conf} ${caddy_conf}.disabled 2>/dev/null || true"},
				{Name: "reload-caddy", Run: "brew services restart caddy || true"},
			},
		},
		"destroy": {
			description: "Destroy WordPress site",
			steps: []pawlStep{
				{Name: "destroy-caddy-conf", Run: "rm -f ${caddy_conf} ${caddy_conf}.disabled"},
				{Name: "destroy-fpm", Run: "rm -f ${fpm_local} ${fpm_pool}"},
				{Name: "destroy-reload", Run: "brew services restart php@${php_ver} 2>/dev/null; brew services restart caddy || true"},
			},
		},
	}

	for name, wf := range workflows {
		cfg := pawlConfig{
			Vars: vars,
			Tasks: map[string]pawlTaskDecl{
				name: {Description: wf.description},
			},
			Workflow: wf.steps,
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(workflowDir, name+".json"), data, 0644); err != nil {
			return err
		}
	}
	return nil
}
