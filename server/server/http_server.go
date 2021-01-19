package server

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
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
	Request string
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	tracer := opentracing.GlobalTracer()
	ectx, err := tracer.Extract(opentracing.HTTPHeaders, r.Header)
	if err != nil {
		log.Println(err)
	}
	log.Printf("%#v", ectx)

	var span opentracing.Span
	if ectx == nil {
		span = tracer.StartSpan("/",
			opentracing.Tag{
				Key:   "user_agent",
				Value: r.UserAgent()},
		)
	} else {
		span = tracer.StartSpan("/", opentracing.ChildOf(ectx),
			opentracing.Tag{
				Key:   "user_agent",
				Value: r.UserAgent()},
		)
	}

	defer span.Finish()

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}

	data := &indexPage{
		Welcome: "Welcome to pepper-poker...",
		Request: string(requestDump),
	}

	file := "index.html"
	tmpl := template.Must(template.ParseFiles(path.Join(*templateDir, file)))

	tmpl.Execute(w, data)
}
