package server

import (
	"fmt"
	"net/http"
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

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// 	before := `
	// <!DOCTYPE html>
	// <html>

	// <head>
	//   <meta charset="utf-8">
	//   <title></title>
	//   <meta name="author" content="">
	//   <meta name="description" content="">
	//   <meta name="viewport" content="width=device-width, initial-scale=1">

	//   <link href="css/style.css" rel="stylesheet">
	// </head>

	// <body>
	// 	`

	// 	after := `
	// </body>

	// </html>
	// `
	// // g.mgr is the game server
	// fmt.Fprintln(w, before)
	// fmt.Fprintf(w, "<h3>pepper-poker server (%s)</h3>", h.mgr.Version())

	// for _, t := range h.mgr.Tables() {
	// 	allinfo, err := t.GetAllTurnLogs()
	// 	if err != nil {
	// 		fmt.Fprintf(w, "error getting turn logs: %v", err)
	// 	}

	// 	for _, info := range allinfo {
	// 		m := jsonpb.Marshaler{
	// 			EmitDefaults: true,
	// 			Indent:       "  ",
	// 		}
	// 		data, _ := m.MarshalToString(info)
	// 		fmt.Fprintln(w, "<p>")
	// 		fmt.Fprintln(w, "<pre>")
	// 		fmt.Fprintf(w, "%v", data)
	// 		fmt.Fprintln(w, "</pre>")
	// 		fmt.Fprintln(w, "</p>")
	// 		fmt.Fprintln(w, "<hr>")
	// 	}
	// }
	// fmt.Fprintln(w, after)

}
