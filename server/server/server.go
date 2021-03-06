package server

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/fatih/color"
	"github.com/fullstorydev/grpcui/standalone"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	channelz "github.com/rantav/go-grpc-channelz"
	"go.opencensus.io/plugin/ochttp"
	"google.golang.org/grpc"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/auth"
)

var (
	httpPort         = flag.String("http_port", "8081", "port to listen on")
	pprofPort        = flag.String("pprof_port", "6060", "port for pprof")
	secureGRPCPort   = flag.String("secure_grpc_port", "8443", "port to listen on for secure grpc")
	insecureGRPCPort = flag.String("insecure_grpc_port", "8082", "port to listen on for insecure grpc")
	grpcUIPort       = flag.String("grpc_ui_port", "8080", "port for serving grpc ui")
	staticDir        = flag.String("static_dir", "server/static/", "location of static http files")

	grpcCrt = flag.String("grpc_crt", "/Users/dant/go/src/github.com/DanTulovsky/pepper-poker-v2/cert/server.crt", "file containg certificate")
	grpcKey = flag.String("grpc_key", "/Users/dant/go/src/github.com/DanTulovsky/pepper-poker-v2/key/server.key", "file containing key")
)

// Server is the poker server
type Server struct {
	secureGRPCServer     *grpc.Server
	insecureGRPCServer   *grpc.Server
	secureGRPCListener   net.Listener
	insecureGRPCListener net.Listener

	authClient *auth.Server
	http       *http.Server

	// Used to send data to the manager on incoming user requests
	managerChan chan actions.PlayerAction

	l *logger.Logger
}

// New returns the server...
func New(tls tls.Certificate, handler http.Handler, secureGRPCPort, insecureGRPCPort, httpPort string, managerChan chan actions.PlayerAction) *Server {

	l := logger.New("server", color.New(color.FgHiGreen))

	secureLis, err := net.Listen("tcp", fmt.Sprintf(":%s", secureGRPCPort))
	if err != nil {
		l.Fatalf("failed to listen (secure): %v", err)
	}

	insecureLis, err := net.Listen("tcp", fmt.Sprintf(":%s", insecureGRPCPort))
	if err != nil {
		l.Fatalf("failed to listen (insecure): %v", err)
	}

	return &Server{
		secureGRPCServer:   secureGRPCServer(tls, auth.NewServerClient(), managerChan),
		insecureGRPCServer: insecureGRPCServer(managerChan),
		http:               httpServer(handler, httpPort),

		secureGRPCListener:   secureLis,
		insecureGRPCListener: insecureLis,

		managerChan: managerChan,

		l: l,
	}
}

// Run runs the server
func Run(ctx context.Context, managerChan chan actions.PlayerAction) error {
	cert, err := tls.LoadX509KeyPair(*grpcCrt, *grpcKey)
	if err != nil {
		return err
	}

	r := mux.NewRouter()

	// Our http  handler
	h := &HTTPHandler{}

	// wrap with OpenCensus handler to provide default http stats
	och := &ochttp.Handler{
		Handler: http.Handler(h),
	}

	// Static files handler
	fs := http.FileServer(http.Dir(*staticDir))

	// HTTP request routing
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	r.PathPrefix("/debug").Handler(channelz.CreateHandler("/debug", fmt.Sprintf(":%s", *insecureGRPCPort)))
	r.PathPrefix("/metrics").Handler(promhttp.Handler())
	r.PathPrefix("/").Handler(och)

	s := New(cert, r, *secureGRPCPort, *insecureGRPCPort, *httpPort, managerChan)

	var wg sync.WaitGroup
	wg.Add(3)

	go s.grpcServe()
	go s.httpServe()
	go s.startPprof()

	go func() {
		if err := s.startGRPCUI(ctx); err != nil {
			s.l.Fatal(err)
		}
	}()

	wg.Wait()

	return nil
}
func (s *Server) startPprof() error {
	pprofMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%s", *pprofPort), pprofMux))

	return nil
}

func (s *Server) startGRPCUI(ctx context.Context) error {

	// embedded grpc ui client
	cc, err := grpc.DialContext(ctx, fmt.Sprintf(":%s", *insecureGRPCPort), grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return err
	}

	grpcui, err := standalone.HandlerViaReflection(ctx, cc, fmt.Sprintf(":%s", *insecureGRPCPort))
	if err != nil {
		return fmt.Errorf("failed on grpcui handler: %v", err)
	}

	s.l.Infof("serving grpc ui on :%s", *grpcUIPort)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", *grpcUIPort))

	if err := http.Serve(listener, grpcui); err != nil {
		log.Fatalf("Failed to serve web UI: %v", err)
	}

	return nil
}

func (s *Server) grpcServe() error {
	s.l.Infof("insecure grpc server ready on port %v", s.insecureGRPCListener.Addr())
	go s.insecureGRPCServer.Serve(s.insecureGRPCListener)

	s.l.Infof("secure grpc server ready on port %v", s.secureGRPCListener.Addr())
	s.l.Info("use /debug/channelz for grpc data")
	return s.secureGRPCServer.Serve(s.secureGRPCListener)
}

func (s *Server) httpServe() error {
	s.l.Infof("http server ready on port %v", s.http.Addr)
	return s.http.ListenAndServe()
}
