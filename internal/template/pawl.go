package template

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yansircc/locwp/internal/site"
)

type pawlConfig struct {
	Vars     map[string]string `json:"vars"`
	Workflow []pawlStep        `json:"workflow"`
}

type pawlStep struct {
	Name   string `json:"name"`
	Run    string `json:"run,omitempty"`
	OnFail string `json:"on_fail,omitempty"`
	Verify string `json:"verify,omitempty"`
}

func WritePawlConfig(path string, sc *site.Config) error {
	cfg := pawlConfig{
		Vars: map[string]string{
			"site":        sc.Name,
			"port":        intToStr(sc.Port),
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
		},
		Workflow: []pawlStep{
			{
				Name: "check-deps",
				Run:  "which php nginx mariadb wp",
			},
			{
				Name:   "create-db",
				Run:    "mariadb -u ${db_user} -e 'CREATE DATABASE IF NOT EXISTS ${db_name}'",
				OnFail: "retry",
			},
			{
				Name:   "download-wp",
				Run:    downloadCmd(),
				OnFail: "retry",
			},
			{
				Name: "gen-wp-config",
				Run:  "php -d memory_limit=512M $(which wp) config create --path=${wp_root} --dbname=${db_name} --dbuser=${db_user} --dbhost=${db_host} --skip-check",
			},
			{
				Name:   "reload-services",
				Run:    "brew services start php@${php_ver} 2>/dev/null; brew services start nginx 2>/dev/null; nginx -s reload",
				OnFail: "retry",
			},
			{
				Name:   "install-wp",
				Run:    "php -d memory_limit=512M $(which wp) core install --path=${wp_root} --url=localhost:${port} --title=${site} --admin_user=${admin_user} --admin_password=${admin_pass} --admin_email=${admin_email}",
				OnFail: "retry",
			},
			{
				Name:   "verify",
				Verify: "manual",
			},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func downloadCmd() string {
	return "php -d memory_limit=512M $(which wp) core download --path=${wp_root} --version=${wp_ver}"
}

func intToStr(n int) string {
	return fmt.Sprintf("%d", n)
}
