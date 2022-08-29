package static

import (
	"embed"
	"net/http"
)

// content holds our static web server content.
//go:embed *
var htmlFiles embed.FS

func StaticPageHandler() http.Handler {
	return http.FileServer(http.FS(htmlFiles))
}
