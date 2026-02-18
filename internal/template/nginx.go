package template

import (
	"fmt"
	"os"

	"github.com/yansircc/locwp/internal/site"
)

func WriteNginxConf(path string, sc *site.Config) error {
	conf := fmt.Sprintf(`server {
    listen %d;
    server_name localhost;
    root %s;
    index index.php index.html;

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
`, sc.Port, sc.WPRoot, sc.SiteDir, sc.SiteDir, sc.Name)

	return os.WriteFile(path, []byte(conf), 0644)
}
