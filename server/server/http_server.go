package server

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"path"
)

var (
	templateDir = flag.String("template_dir", "server/templates/", "html template dir")
)

func httpServer(handler http.Handler, port string) *http.Server {
	httpServer := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf(":%s", port),
	}

	return httpServer
}

// HTTPHandler handles http traffic
type HTTPHandler struct {
}

type indexPage struct {
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "text/html")

	data := &indexPage{}

	file := "index.html"
	tmpl := template.Must(template.ParseFiles(path.Join(*templateDir, file)))

	tmpl.Execute(w, data)
}
