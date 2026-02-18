package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yansircc/locwp/internal/site"
)

type pawlConfig struct {
	Vars     map[string]string          `json:"vars"`
	Tasks    map[string]pawlTaskDecl    `json:"tasks"`
	Workflow []pawlStep                 `json:"workflow"`
}

type pawlTaskDecl struct {
	Description string `json:"description,omitempty"`
}

type pawlStep struct {
	Name   string `json:"name"`
	Run    string `json:"run,omitempty"`
	OnFail string `json:"on_fail,omitempty"`
}

// WritePawlWorkflows generates all lifecycle workflow files under workflowDir.
func WritePawlWorkflows(workflowDir string, sc *site.Config) error {
	baseDir := filepath.Dir(filepath.Dir(sc.SiteDir))
	vhost := filepath.Join(baseDir, "nginx", "sites", sc.Name+".conf")

	vars := map[string]string{
		"site":        sc.Name,
		"port":        fmt.Sprintf("%d", sc.Port),
		"db_name":     sc.DBName,
		"db_user":     sc.DBUser,
		"db_host":     sc.DBHost,
		"wp_root":     sc.WPRoot,
		"wp_ver":      sc.WPVer,
		"php_ver":     sc.PHP,
		"site_dir":    sc.SiteDir,
		"admin_user":  sc.AdminUser,
		"admin_pass":  sc.AdminPass,
		"admin_email": sc.AdminEmail,
		"vhost":       vhost,
		"nginx_link":  filepath.Join(HomebrewPrefix(), "etc", "nginx", "servers", "locwp-"+sc.Name+".conf"),
		"fpm_local":   filepath.Join(baseDir, "php", sc.Name+".conf"),
		"fpm_pool":    filepath.Join(FPMPoolDir(sc.PHP), "locwp-"+sc.Name+".conf"),
	}

	type workflowDef struct {
		description string
		steps       []pawlStep
	}

	workflows := map[string]workflowDef{
		"provision": {
			description: "Provision WordPress site",
			steps: []pawlStep{
				{Name: "check-deps", Run: "which php nginx mariadb wp"},
				{Name: "create-db", Run: "mariadb -u ${db_user} -e 'CREATE DATABASE IF NOT EXISTS `${db_name}`'", OnFail: "retry"},
				{Name: "download-wp", Run: "php -d memory_limit=512M $(which wp) core download --path=${wp_root} --version=${wp_ver}", OnFail: "retry"},
				{Name: "gen-wp-config", Run: "php -d memory_limit=512M $(which wp) config create --path=${wp_root} --dbname=${db_name} --dbuser=${db_user} --dbhost=${db_host} --skip-check"},
				{Name: "provision-services", Run: "brew services restart php@${php_ver} 2>/dev/null; brew services start nginx 2>/dev/null; nginx -s reload", OnFail: "retry"},
				{Name: "install-wp", Run: "php -d memory_limit=512M $(which wp) core install --path=${wp_root} --url=localhost:${port} --title=${site} --admin_user=${admin_user} --admin_password=${admin_pass} --admin_email=${admin_email}", OnFail: "retry"},
				{Name: "set-permalinks", Run: "php -d memory_limit=512M $(which wp) rewrite structure '/%postname%/' --path=${wp_root} && php -d memory_limit=512M $(which wp) rewrite flush --path=${wp_root}"},
			},
		},
		"start": {
			description: "Start WordPress site",
			steps: []pawlStep{
				{Name: "enable-vhost", Run: "mv ${vhost}.disabled ${vhost} 2>/dev/null || true"},
				{Name: "start-php", Run: "brew services start php@${php_ver}"},
				{Name: "start-nginx", Run: "nginx -s reload"},
			},
		},
		"stop": {
			description: "Stop WordPress site",
			steps: []pawlStep{
				{Name: "disable-vhost", Run: "mv ${vhost} ${vhost}.disabled 2>/dev/null || true"},
				{Name: "stop-nginx", Run: "nginx -s reload"},
			},
		},
		"destroy": {
			description: "Destroy WordPress site",
			steps: []pawlStep{
				{Name: "drop-db", Run: "mariadb -u ${db_user} -e 'DROP DATABASE IF EXISTS `${db_name}`'"},
				{Name: "destroy-vhost", Run: "rm -f ${vhost} ${vhost}.disabled ${nginx_link}"},
				{Name: "destroy-fpm", Run: "rm -f ${fpm_local} ${fpm_pool}"},
				{Name: "destroy-reload", Run: "brew services restart php@${php_ver} 2>/dev/null; nginx -s reload"},
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
