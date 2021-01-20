package server

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"path"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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

	ectx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		log.Println(err)
	}
	spew.Dump(ectx)

	span := opentracing.StartSpan("/",
		ext.RPCServerOption(ectx),
		opentracing.Tag{
			Key:   "user_agent",
			Value: r.UserAgent()},
	)
	spew.Dump(span)

	defer span.Finish()

	// If sending RPC to a downstream service, use this context
	// ctx := opentracing.ContextWithSpan(context.Background(), serverSpan)

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
