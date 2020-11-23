package main

import (
	"context"
	"flag"
	"io"
	"math/rand"
	"time"

	"github.com/DanTulovsky/logger"

	"github.com/fatih/color"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	grpcCrt    = flag.String("grpc_crt", "../../cert/server.crt", "file containg certificate")
	httpPort   = flag.String("http_port", "", "port to listen on, random if empty")
	serverAddr = flag.String("server_address", "localhost:8082", "tls server address and port")
)

func main() {

	flag.Parse()
	logg := logger.New("client", color.New(color.FgBlue))

	logg.Info("Starting client...")

	ctx := context.Background()
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}

	var conn *grpc.ClientConn
	var err error
	if conn, err = grpc.Dial(*serverAddr, opts...); err != nil {
		logg.Fatal(err)
	}
	client := ppb.NewPokerServerClient(conn)

	ctxCancel, cancel := context.WithCancel(ctx)

	stream, err := client.Play(ctxCancel)
	if err != nil {
		logg.Fatal(err)
	}

	waitc := make(chan bool)
	// receive server messages
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				cancel()
				logg.Fatal("error receiving from server")
			}
			logg.Infof("Received from server: %v", in.Output)
		}
	}()

	// register
	logg.Info("Registering...")
	req := &ppb.ClientData{
		PlayerID:     "1",
		PlayerName:   "Bob",
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	if err := stream.Send(req); err != nil {
		logg.Fatal(err)
	}

	// join table
	logg.Info("Joining table...")
	req = &ppb.ClientData{
		PlayerID:     "1",
		PlayerName:   "Bob",
		PlayerAction: ppb.PlayerAction_PlayerActionJoinTable,
	}
	if err := stream.Send(req); err != nil {
		logg.Fatal(err)
	}

	// send data after
	logg.Info("Feeding data...")
	for {
		req := &ppb.ClientData{
			Input:        rand.Int63n(200) + 100,
			PlayerAction: ppb.PlayerAction_PlayerActionRandomInt,
		}
		if err := stream.Send(req); err != nil {
			logg.Fatal(err)
		}

		time.Sleep(time.Second * 3)
	}
}
