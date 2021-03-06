package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/fatih/color"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/auth"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

func insecureGRPCServer(managerChan chan actions.PlayerAction) *grpc.Server {

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			// grpc_auth.StreamServerInterceptor(pokerAuthFunc),
			// grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(opentracing.GlobalTracer())),
			otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer()),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			// grpc_auth.UnaryServerInterceptor(pokerAuthFunc),
			// grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(opentracing.GlobalTracer())),
			otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer()),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}

	insecureServer := grpc.NewServer(opts...)
	ps := newPokerServer("insecure", managerChan)
	ppb.RegisterPokerServerServer(insecureServer, ps)
	reflection.Register(insecureServer)

	service.RegisterChannelzServiceToServer(insecureServer)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("grpc.health.v1.helloservice", 1)
	grpc_health_v1.RegisterHealthServer(insecureServer, healthServer)

	grpc_prometheus.Register(insecureServer)
	grpc_prometheus.EnableHandlingTimeHistogram()
	return insecureServer
}

func secureGRPCServer(cert tls.Certificate, authClient *auth.Server, managerChan chan actions.PlayerAction) *grpc.Server {

	recoveryOpts := []grpc_recovery.Option{
		// grpc_recovery.WithRecoveryHandler(customFunc),
	}

	opts := []grpc.ServerOption{
		// Enable TLS for all incoming connections.
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_auth.StreamServerInterceptor(authClient.PokerAuthFunc),
			// grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(opentracing.GlobalTracer())),
			otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer()),
			grpc_prometheus.StreamServerInterceptor,
			grpc_recovery.StreamServerInterceptor(recoveryOpts...),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_auth.UnaryServerInterceptor(authClient.PokerAuthFunc),
			// grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(opentracing.GlobalTracer())),
			otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer()),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_recovery.UnaryServerInterceptor(recoveryOpts...),
		)),
	}

	secureServer := grpc.NewServer(opts...)
	ps := newPokerServer("secure", managerChan)
	ppb.RegisterPokerServerServer(secureServer, ps)
	reflection.Register(secureServer)

	service.RegisterChannelzServiceToServer(secureServer)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("grpc.health.v1.helloservice", 1)
	grpc_health_v1.RegisterHealthServer(secureServer, healthServer)

	grpc_prometheus.Register(secureServer)
	grpc_prometheus.EnableHandlingTimeHistogram()
	return secureServer
}

func newPokerServer(name string, managerChan chan actions.PlayerAction) *pokerServer {
	return &pokerServer{
		name:        name,
		managerChan: managerChan,
		l:           logger.New(fmt.Sprintf("%v grpc_handler", name), color.New(color.FgMagenta)),
	}
}

// pokerServer is the grpc server
type pokerServer struct {
	name string

	// used to send data to the manager
	managerChan chan actions.PlayerAction

	l *logger.Logger
}

// Register registers with the server
func (ps *pokerServer) Register(ctx context.Context, in *ppb.RegisterRequest) (*ppb.RegisterResponse, error) {
	ps.l.Info("Received Register RPC")

	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	// Username is set by the authentication library into the context
	cinfo := in.GetClientInfo()
	cinfo.PlayerUsername = *ctx.Value(auth.UinfoType("uinfo")).(*gocloak.UserInfo).PreferredUsername

	resultc := make(chan actions.PlayerActionResult)
	action := actions.NewPlayerAction(ctx, ppb.PlayerAction_PlayerActionRegister, nil, in.GetClientInfo(), nil, resultc)

	// Send request to manager
	ps.managerChan <- action

	// block on response
	res := <-resultc
	if res.Err != nil {
		if errors.Is(res.Err, actions.ErrUserExists) {
			return nil, status.Errorf(codes.AlreadyExists, "%v", res.Err)
		}
		return nil, status.Errorf(codes.Unknown, "invalid request: %v", res.Err)
	}

	out := res.Result.(*ppb.RegisterResponse)

	return out, err
}

// JoinTable joins a table
func (ps *pokerServer) JoinTable(ctx context.Context, in *ppb.JoinTableRequest) (*ppb.JoinTableResponse, error) {
	ps.l.Info("Received JoinTable RPC")

	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	cinfo := in.GetClientInfo()
	cinfo.PlayerUsername = *ctx.Value(auth.UinfoType("uinfo")).(*gocloak.UserInfo).PreferredUsername

	resultc := make(chan actions.PlayerActionResult)
	action := actions.NewPlayerAction(ctx, ppb.PlayerAction_PlayerActionJoinTable, nil, in.GetClientInfo(), nil, resultc)

	// Send request to manager
	ps.managerChan <- action

	// block on response
	res := <-resultc
	if res.Err != nil {
		return nil, fmt.Errorf("invalid request: %v", res.Err)
	}

	out := res.Result.(*ppb.JoinTableResponse)

	return out, err
}

// TakeTurn takes a single poker turn
func (ps *pokerServer) TakeTurn(ctx context.Context, in *ppb.TakeTurnRequest) (*ppb.TakeTurnResponse, error) {
	ps.l.Info("Received TakeTurn RPC")

	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	cinfo := in.GetClientInfo()
	cinfo.PlayerUsername = *ctx.Value(auth.UinfoType("uinfo")).(*gocloak.UserInfo).PreferredUsername

	resultc := make(chan actions.PlayerActionResult)
	action := actions.NewPlayerAction(ctx, in.GetPlayerAction(), in.GetActionOpts(), in.GetClientInfo(), nil, resultc)

	// Send request to manager
	ps.managerChan <- action

	// block on response
	res := <-resultc
	if res.Err != nil {
		return nil, fmt.Errorf("invalid request: %v", res.Err)
	}

	out := res.Result.(*ppb.TakeTurnResponse)
	return out, err
}

func (ps *pokerServer) AckToken(ctx context.Context, in *ppb.AckTokenRequest) (*ppb.AckTokenResponse, error) {
	ps.l.Info("Received AckToken RPC")

	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	cinfo := in.GetClientInfo()
	cinfo.PlayerUsername = *ctx.Value(auth.UinfoType("uinfo")).(*gocloak.UserInfo).PreferredUsername

	resultc := make(chan actions.PlayerActionResult)
	opts := &ppb.ActionOpts{
		AckToken: in.GetToken(),
	}
	action := actions.NewPlayerAction(ctx, ppb.PlayerAction_PlayerActionAckToken, opts, in.GetClientInfo(), nil, resultc)

	// Send request to manager
	ps.managerChan <- action

	// block on response
	res := <-resultc
	if res.Err != nil {
		return nil, fmt.Errorf("invalid request: %v", res.Err)
	}

	out := res.Result.(*ppb.AckTokenResponse)
	return out, err
}

// Play is a server streaming RPC that us used to send GameData back to the client as needed
func (ps *pokerServer) Play(in *ppb.PlayRequest, stream ppb.PokerServer_PlayServer) error {
	ps.l.Info("Received Play RPC")
	ctx := stream.Context()

	cinfo := in.GetClientInfo()
	cinfo.PlayerUsername = *ctx.Value(auth.UinfoType("uinfo")).(*gocloak.UserInfo).PreferredUsername

	// Create a channel that the game can send data back to the client on
	// it is read in the goroutine started below
	toPlayerC := make(chan actions.GameData)

	resultc := make(chan actions.PlayerActionResult)
	action := actions.NewPlayerAction(stream.Context(), ppb.PlayerAction_PlayerActionPlay, nil, in.GetClientInfo(), toPlayerC, resultc)

	// Send request to manager
	ps.managerChan <- action

	// block on response, an error here means we failed to subscribe and should exit
	res := <-resultc
	if res.Err != nil {
		return fmt.Errorf("invalid request: %v", res.Err)
	}

	// start a goroutine to send data back to client
	// the fromManagerChan get attached to the player itself and allows
	// anything that has access to the player object to send updates
	var err error
OUTER:
	for {
		select {
		case input, ok := <-toPlayerC:

			// span, _ := opentracing.StartSpanFromContext(ctx, "sendPlayerUpdate")
			span := opentracing.StartSpan("sendPlayerUpdate",
				opentracing.FollowsFrom(opentracing.SpanFromContext(ctx).Context()))
			span.SetTag("playerUsername", cinfo.GetPlayerUsername())
			span.SetTag("waitTurnName", input.Data.GetWaitTurnName())
			span.SetTag("waitTurnNum", input.Data.GetWaitTurnNum())
			span.SetTag("waitTurnTimeLeftSec", input.Data.GetWaitTurnTimeLeftSec())
			span.SetTag("waitTurnTimeMaxSec", input.Data.GetWaitTurnTimeMaxSec())
			span.SetTag("tableName", input.Data.GetInfo().GetTableName())
			ext.Component.Set(span, "grpc_server")

			if !ok {
				ps.l.Debug("Lost connection to table player channel")
				err = fmt.Errorf("Lost connection to table player channel")

				ext.Error.Set(span, true)
				span.LogFields(log.String("error", err.Error()))
				span.Finish()
				break OUTER
			}

			ps.l.Debugf("Sending data to client (%v): %#v", in.ClientInfo.GetPlayerUsername(), input.Data.GetWaitTurnName())
			if err = stream.Send(input.Data); err != nil {
				message := fmt.Sprintf("client connection to %v lost: %v", in.ClientInfo.PlayerUsername, err)
				ps.l.Infof(message)

				ext.Error.Set(span, true)
				span.LogFields(log.String("error", message))
				span.Finish()
				break OUTER
			}
			ps.l.Debugf("Sent data to client (%v): %#v", in.ClientInfo.PlayerUsername, input.Data.WaitTurnID)

			span.Finish()
		}
	}

	// Return any player.Stack() to player.Bank()
	if err := ps.playerDisconnected(stream.Context(), cinfo); err != nil {
		ps.l.Error(err)
	}

	ps.l.Info("Client channel closed, exiting thread...")
	return err
}

func (ps *pokerServer) playerDisconnected(ctx context.Context, cinfo *ppb.ClientInfo) error {

	span, _ := opentracing.StartSpanFromContext(ctx, "playerDisconnected")
	span.SetTag("playerUsername", cinfo.PlayerUsername)
	ext.Component.Set(span, "grpc_server")
	defer span.Finish()

	resultc := make(chan actions.PlayerActionResult)
	action := actions.NewPlayerAction(ctx, ppb.PlayerAction_PlayerActionDisconnect, nil, cinfo, nil, resultc)

	// Send request to manager
	ps.managerChan <- action

	// block on response, an error here means we failed to subscribe and should exit
	res := <-resultc
	if res.Err != nil {
		ext.Error.Set(span, true)
		span.LogFields(log.String("error", res.Err.Error()))
		return fmt.Errorf("invalid request: %v", res.Err)
	}

	return nil
}
