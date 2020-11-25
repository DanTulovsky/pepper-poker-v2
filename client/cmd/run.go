package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/DanTulovsky/logger"
	"github.com/Pallinder/go-randomdata"

	"github.com/fatih/color"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"

	"github.com/DanTulovsky/pepper-poker-v2/id"
	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	grpcCrt    = flag.String("grpc_crt", "../../cert/server.crt", "file containg certificate")
	httpPort   = flag.String("http_port", "", "port to listen on, random if empty")
	serverAddr = flag.String("server_address", "localhost:8082", "tls server address and port")

	name = flag.String("name", randomdata.SillyName(), "player name")

	playerID  id.PlayerID
	tableID   id.TableID
	gameState ppb.GameState
)

func main() {

	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	logg := logger.New(fmt.Sprintf("client [%v]", *name), color.New(color.FgBlue))

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

	// register
	logg.Info("Registering...")
	req := &ppb.RegisterRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: *name,
		},
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	var res *ppb.RegisterResponse
	if res, err = client.Register(ctx, req); err != nil {
		logg.Fatal(err)
	}
	playerID = id.PlayerID(res.GetPlayerID())
	logg.Debugf("playerID: %v", playerID)

	// join table
	logg.Info("Joining table...")
	reqJT := &ppb.JoinTableRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: *name,
			PlayerID:   playerID.String(),
			TableID:    tableID.String(),
		},
		PlayerAction: ppb.PlayerAction_PlayerActionJoinTable,
	}

	var resJT *ppb.JoinTableResponse
	if resJT, err = client.JoinTable(ctx, reqJT); err != nil {
		logg.Fatal(err)
	}
	tableID = id.TableID(resJT.GetTableID())
	logg.Debugf("tableID: %v", tableID)

	// Subscribe to GameData from the server after joing table
	reqPlay := &ppb.PlayRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: *name,
			PlayerID:   playerID.String(),
			TableID:    tableID.String(),
		},
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	stream, err := client.Play(ctxCancel, reqPlay)
	if err != nil {
		logg.Fatal(err)
	}

	// send server response on this channel to process in the main thread
	datac := make(chan *ppb.GameData)
	donec := make(chan bool)
	// receive server messages
	go func() {
		for {
			logg.Debug("waiting for server data...")
			in, err := stream.Recv()
			logg.Debug("received server data...")
			if err == io.EOF {
				logg.Debugf("EOF received from server, exiting GameData thread")
				return
			}
			if err != nil {
				cancel()
				logg.Fatal("error receiving from server")
			}
			// send the server message to main thread for processing
			logg.Debug("sending server data to main thread...")
			datac <- in
		}
	}()

	// Receive GameData on datac channel and act on it
OUTER:
	for {
		logg.Debug("Waiting for GameData...")
		// process server messages if any (on datac channel)
		select {
		case in := <-datac:
			logg.Debug("received game data in main thread")

			if playerID != id.PlayerID(in.PlayerID) {
				logg.Fatal("Mismatch in playerID; expected: %v; got: %v", playerID, id.PlayerID(in.PlayerID))
			}
			if tableID != id.TableID(in.GetInfo().GetTableID()) {
				logg.Fatalf("Mismatch in tableID; expected: %v; got: %v", tableID, id.TableID(in.GetInfo().GetTableID()))
			}

			waitID := id.PlayerID(in.WaitTurnID)
			gameState = in.GetInfo().GetGameState()

			logg.Infof("Current Turn playerID: %v", in.WaitTurnID)
			logg.Infof("Current State: %v", gameState)

			if gameState == ppb.GameState_GameStateFinished {
				logg.Info("Game Finished!")
				donec <- true
				conn.Close()
			}

			if playerID == waitID {
				action := ppb.PlayerAction_PlayerActionCheck
				logg.Infof("Taking Turn: %v", action)

				req := &ppb.TakeTurnRequest{
					ClientInfo: &ppb.ClientInfo{
						PlayerName: *name,
						PlayerID:   playerID.String(),
						TableID:    tableID.String(),
					},
					PlayerAction: action,
				}
				_, err := client.TakeTurn(ctx, req)
				if err != nil {
					logg.Error(err)
				}
				time.Sleep(time.Second * 1)
			}
		case <-donec:
			logg.Info("Exiting server data receiver as requested...")
			break OUTER
		}
	}
}
