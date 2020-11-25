package pokerclient

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opencensus.io/plugin/ochttp"
)

// Server is the poker client debug server
type Server struct {
	http *http.Server
	pc   *PokerClient
}

// NewServer returns the server...
func NewServer(pc *PokerClient, handler http.Handler, httpPort string) *Server {
	return &Server{
		http: httpServer(handler, httpPort),
		pc:   pc,
	}
}

func (s *Server) httpServe() error {
	log.Printf("http server ready http://%v", s.http.Addr)
	return s.http.ListenAndServe()
}

// RunServer runs the server
func RunServer(ctx context.Context, pc *PokerClient, httpPort string) error {
	log.Printf("starting server...")

	r := mux.NewRouter()

	// Our root http  handler
	h := &HTTPHandler{
		pc: pc,
	}

	// wrap with OpenCensus handler to provide default http stats
	och := &ochttp.Handler{
		Handler: http.Handler(h),
	}

	r.PathPrefix("/metrics").Handler(promhttp.Handler())
	r.PathPrefix("/").Handler(och)

	s := NewServer(pc, r, httpPort)

	var wg sync.WaitGroup
	wg.Add(2)

	go s.httpServe()

	log.Println("Waiting for servers to exit...")
	wg.Wait()

	return nil
}
