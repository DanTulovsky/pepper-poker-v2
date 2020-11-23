package server

import (
	"crypto/tls"
	"io"

	"github.com/fatih/color"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

func insecureGRPCServer(managerChan chan actions.PlayerAction) *grpc.Server {

	opts := []grpc.ServerOption{
		// The following grpc.ServerOption adds an interceptor for all unary
		// RPCs. To configure an interceptor for streaming RPCs, see:
		// https://godoc.org/google.golang.org/grpc#StreamInterceptor
		// Enable TLS for all incoming connections.
		// grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_opentracing.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}

	insecureServer := grpc.NewServer(opts...)
	pokerServer := newPokerServer(managerChan)
	ppb.RegisterPokerServerServer(insecureServer, pokerServer)
	reflection.Register(insecureServer)

	service.RegisterChannelzServiceToServer(insecureServer)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("grpc.health.v1.helloservice", 1)
	grpc_health_v1.RegisterHealthServer(insecureServer, healthServer)

	grpc_prometheus.Register(insecureServer)
	grpc_prometheus.EnableHandlingTimeHistogram()
	return insecureServer
}

func secureGRPCServer(cert tls.Certificate, managerChan chan actions.PlayerAction) *grpc.Server {

	opts := []grpc.ServerOption{
		// The following grpc.ServerOption adds an interceptor for all unary
		// RPCs. To configure an interceptor for streaming RPCs, see:
		// https://godoc.org/google.golang.org/grpc#StreamInterceptor
		// Enable TLS for all incoming connections.
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_opentracing.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}

	secureServer := grpc.NewServer(opts...)
	pokerServer := newPokerServer(managerChan)
	ppb.RegisterPokerServerServer(secureServer, pokerServer)
	reflection.Register(secureServer)

	service.RegisterChannelzServiceToServer(secureServer)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("grpc.health.v1.helloservice", 1)
	grpc_health_v1.RegisterHealthServer(secureServer, healthServer)

	grpc_prometheus.Register(secureServer)
	grpc_prometheus.EnableHandlingTimeHistogram()
	return secureServer
}

func newPokerServer(managerChan chan actions.PlayerAction) *pokerServer {
	return &pokerServer{
		managerChan: managerChan,
		l:           logger.New("grpc_handler", color.New(color.FgMagenta)),
	}
}

// pokerServer is the grpc server
type pokerServer struct {
	// used to send data to the manager
	managerChan chan actions.PlayerAction

	l *logger.Logger
}

func (ps *pokerServer) Play(stream ppb.PokerServer_PlayServer) error {

	// Create a channel for manager to send updates on
	fromManagerChan := make(chan actions.ManagerAction)

	// start a goroutine to send data back to client
	go func() {

		ps.l.Info("This will listen and send data back to client")
		for {
			select {
			case in := <-fromManagerChan:
				ps.l.Infof("Sending data to client: %v", in)
				res := &ppb.GameData{
					Output: in.Result,
				}
				stream.Send(res)
			default:
			}
		}

	}()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		action := actions.NewPlayerAction(in, fromManagerChan)

		ps.managerChan <- action

	}
}
