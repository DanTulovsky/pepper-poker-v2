package server

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"path"

	"github.com/opentracing/opentracing-go"
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
	Welcome string
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	parentSpan := opentracing.SpanFromContext(r.Context())

	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"/",
		opentracing.ChildOf(parentSpan.Context()),
		opentracing.Tag{
			Key:   "user_agent",
			Value: r.UserAgent()},
	)
	defer span.Finish()

	data := &indexPage{
		Welcome: "Welcome to pepper-poker...",
	}

	file := "index.html"
	tmpl := template.Must(template.ParseFiles(path.Join(*templateDir, file)))

	tmpl.Execute(w, data)
}
