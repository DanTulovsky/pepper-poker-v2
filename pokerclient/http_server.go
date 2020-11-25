package pokerclient

import (
	"fmt"
	"net/http"
)

func httpServer(handler http.Handler, port string) *http.Server {
	httpServer := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
	}

	return httpServer
}

// HTTPHandler handles http traffic
type HTTPHandler struct {
	pc *PokerClient
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
	// 	`
	// 	// h.pc is the pokerClient
	// 	fmt.Fprintln(w, before)
	// 	fmt.Fprintln(w, "<h3>pepper-poker client</h3>")

	// 	m := jsonpb.Marshaler{
	// 		EmitDefaults: true,
	// 		Indent:       "  ",
	// 	}
	// 	data, _ := m.MarshalToString(h.pc.TableInfo.Current())
	// 	fmt.Fprintln(w, "<h4>Client Info</h4>")
	// 	fmt.Fprintf(w, "Name: %v<br>", h.pc.Name)
	// 	fmt.Fprintf(w, "Player ID: %v<br>", h.pc.PlayerID)
	// 	fmt.Fprintf(w, "Table ID: %v<br>", h.pc.TableID)
	// 	fmt.Fprintf(w, "Round ID: %v<br>", h.pc.RoundID)
	// 	fmt.Fprintln(w, "<p>")
	// 	fmt.Fprintln(w, "<h4>tableInfo</h4>")
	// 	fmt.Fprintln(w, "<pre>")
	// 	fmt.Fprintf(w, "%v", data)
	// 	fmt.Fprintln(w, "</pre>")
	// 	fmt.Fprintln(w, "</p>")
	// 	fmt.Fprintln(w, "<hr>")

	// 	fmt.Fprintln(w, after)
}
