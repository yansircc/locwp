package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yansircc/locwp/internal/config"
	"github.com/yansircc/locwp/internal/site"
)

func WriteNginxConf(path string, sc *site.Config) error {
	sslDir := config.SSLDir()
	conf := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name %s;
    root %s;
    index index.php index.html;

    ssl_certificate     %s;
    ssl_certificate_key %s;

    access_log %s/logs/access.log;
    error_log  %s/logs/error.log;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/tmp/locwp-%s.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires max;
        log_not_found off;
    }
}
`, sc.Domain, sc.Domain, sc.WPRoot,
		filepath.Join(sslDir, "_wildcard.local.pem"),
		filepath.Join(sslDir, "_wildcard.local-key.pem"),
		sc.SiteDir, sc.SiteDir, sc.Name)

	return os.WriteFile(path, []byte(conf), 0644)
}
