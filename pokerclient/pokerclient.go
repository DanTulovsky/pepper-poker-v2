package pokerclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/phayes/freeport"
	"github.com/tcnksm/go-input"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	imgcat "github.com/martinlindhe/imgcat/lib"

	"github.com/DanTulovsky/deck"
	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/actions"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	grpcCrt            = flag.String("grpc_crt", "../../../cert/server.crt", "file containg certificate")
	httpPort           = flag.String("http_port", "", "port to listen on, random if empty")
	secureServerAddr   = flag.String("server_address", "localhost:8443", "tls server address and port")
	insecureServerAddr = flag.String("insecure_server_address", "localhost:8082", "insecure server address and port")

	ui *input.UI = &input.UI{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}
)

const (
	game    string = "Pepper-Poker"
	version string = "0.1-pre-alpha"
)

// PokerClient is the poker client
type PokerClient struct {
	Name     string
	PlayerID id.PlayerID
	TableID  id.TableID
	position int64
	client   ppb.PokerServerClient

	// The background GameData goroutine sends server updates on this channel
	datac chan *ppb.GameData

	// the last acked token
	lastAckedToken string

	gameState ppb.GameState

	conn   *grpc.ClientConn
	cancel context.CancelFunc

	stopGameDataStreaming chan bool

	// communication channel between the caller and this client
	action       chan *actions.PlayerAction
	actionResult chan *actions.PlayerActionResult
	inputWanted  chan *ppb.GameData

	l *logger.Logger
}

// New returns a new pokerClient
func New(ctx context.Context, name string, insecure bool, actions chan *actions.PlayerAction, actionResult chan *actions.PlayerActionResult, inputWanted chan *ppb.GameData) (*PokerClient, error) {
	// showWelcome()

	rand.Seed(time.Now().UnixNano())

	if *httpPort == "" {
		port, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		*httpPort = fmt.Sprintf("%d", port)
	}

	logger := logger.New(name, color.New(color.FgGreen))
	pc := &PokerClient{
		Name:         name,
		l:            logger,
		action:       actions,
		actionResult: actionResult,
		inputWanted:  inputWanted,
		datac:        make(chan *ppb.GameData),
	}

	opts := []grpc.DialOption{

		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}

	grpc_prometheus.EnableClientHandlingTimeHistogram()
	serverAdd := *insecureServerAddr

	if !insecure {
		var err error
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			return nil, err
		}

		opts = append(opts, grpc.WithTransportCredentials(tlsCredentials))
		serverAdd = *secureServerAddr
	} else {
		logger.Warn("Using an insecure connection to the server!")
		opts = append(opts, grpc.WithInsecure())
	}

	var err error
	if pc.conn, err = grpc.Dial(serverAdd, opts...); err != nil {
		return nil, err
	}
	pc.client = ppb.NewPokerServerClient(pc.conn)

	// http server for statusz
	go RunServer(ctx, pc, *httpPort)

	return pc, nil
}

// Reset resets the client for the next hand
func (pc *PokerClient) Reset() {
	pc.l.Info("Resetting PokerClient for next game...")

	pc.lastAckedToken = ""
	pc.cancel() // TODO: Needed?
}

// Play is called after joining table to begin streaming GameData
func (pc *PokerClient) Play(ctx context.Context, donec chan bool, handDone chan bool, errc chan error) {
	pc.l.Info("Starting GameData streamer..")

	ctxCancel, cancel := context.WithCancel(ctx)
	pc.cancel = cancel

	// Subscribe to GameData from the server after joing table
	reqPlay := &ppb.PlayRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	stream, err := pc.client.Play(ctxCancel, reqPlay)
	if err != nil {
		errc <- err
		return
	}

	go pc.receiveGameData(stream, donec)

	if err := pc.processGameData(ctx); err != nil {
		errc <- err
	}
}

// processGameData receives GameData on the channel and acts on it
func (pc *PokerClient) processGameData(ctx context.Context) error {

	// Receive GameData on datac channel and act on it
	for {
		pc.l.Debug("Waiting for GameData...")
		// process server messages if any (on datac channel)
		select {
		case in := <-pc.datac:
			pc.l.Debug("received game data in main thread")

			if pc.PlayerID != id.PlayerID(in.PlayerID) {
				pc.l.Fatal("Mismatch in playerID; expected: %v; got: %v", pc.PlayerID, id.PlayerID(in.PlayerID))
			}
			if pc.TableID != id.TableID(in.GetInfo().GetTableID()) {
				pc.l.Fatalf("Mismatch in tableID; expected: %v; got: %v", pc.TableID, id.TableID(in.GetInfo().GetTableID()))
			}

			waitID := id.PlayerID(in.WaitTurnID)
			pc.gameState = in.GetInfo().GetGameState()
			ackToken := in.GetInfo().GetAckToken()

			pc.l.Infof("Current Turn playerID: %v", in.WaitTurnID)
			pc.l.Infof("Current State: %v", pc.gameState)

			if pc.gameState == ppb.GameState_GameStateFinished {
				pc.l.Info("Game Finished!")
				pc.conn.Close()
				time.Sleep(time.Second * 2)
				os.Exit(0)
			}

			if ackToken != pc.lastAckedToken && ackToken != "" {
				pc.l.Infof("Acking [%v]", ackToken)
				pc.Ack(ctx, ackToken)
			}

			if pc.PlayerID == waitID {
				pc.TakeTurn(ctx, in)
				// action := ppb.PlayerAction_PlayerActionCheck
				// pc.logg.Infof("Taking Turn: %v", action)

				// req := &ppb.TakeTurnRequest{
				// 	ClientInfo: &ppb.ClientInfo{
				// 		PlayerName: pc.Name,
				// 		PlayerID:   pc.PlayerID.String(),
				// 		TableID:    pc.TableID.String(),
				// 	},
				// 	PlayerAction: action,
				// }
				// _, err := pc.client.TakeTurn(ctx, req)
				// if err != nil {
				// 	pc.l.Error(err)
				// }
				// time.Sleep(timeSecond * 1)
			}
		}
	}
}

// ReceiveGameData receives GameData from the server and sends it to the main thread over a channel
func (pc *PokerClient) receiveGameData(stream ppb.PokerServer_PlayClient, donec chan bool) error {
	pc.l.Debug("Started eceive GameData thread...")

OUTER:
	for {
		select {
		case <-donec:
			pc.l.Info("calling cancel on server stream (stop called)")
			// cancel()
			break OUTER
		default:
		}

		pc.l.Debug("waiting for server data...")
		in, err := stream.Recv()
		pc.l.Debug("received server data...")
		if err == io.EOF {
			return fmt.Errorf("EOF received from server, exiting GameData thread")
		}
		if err != nil {
			// pc.cancel()
			pc.l.Fatal("error receiving from server")
		}
		// send the server message to main thread for processing
		pc.l.Debug("sending server data to main thread...")
		pc.datac <- in
	}

	return nil
}

// TakeTurn takes a turn
func (pc *PokerClient) TakeTurn(ctx context.Context, in *ppb.GameData) error {
	pc.l.Debug("Trying to take my turn...")

	// if !pc.IsMyTurn() {
	// 	next, err := pc.WhoseTurn()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if pc.TableReady() {
	// 		left := pc.TurnLog.TurnTimeLeft()
	// 		if pc.Name != next.GetName() {
	// 			pc.l.Debugf("[%v] Waiting for %v to move... (%v)", pc.Name, next.GetName(), left)
	// 		}
	// 	} else {
	// 		startIn := pc.TableInfo.GameStartsIn()
	// 		pc.l.Debugf("[%v] Game starts in: %v", pc.Name, startIn)
	// 	}
	// 	return nil
	// }

	// pc.l.Infof("[%v] taking my turn...", pc.Name)
	return pc.processTurn(ctx, in)
}

// processTurn executes this client's turn
func (pc *PokerClient) processTurn(ctx context.Context, in *ppb.GameData) error {
	// Show most up to date status from the server
	// pc.showGameState()

	// pc.showCards(append(pc.myCards(), deck.CardsFromProto(pc.TurnLog.CommunityCards().Card)...), true)

	// Tell user input is needed
	pc.inputWanted <- in

	// block until we get a result
	paction := <-pc.action

	var err error

	switch paction.Action {
	case ppb.PlayerAction_PlayerActionCall:
		if err = pc.Call(ctx); err != nil {
			pc.l.Infof("error calling: %v", err)
		}
	case ppb.PlayerAction_PlayerActionCheck:
		if err = pc.Check(ctx); err != nil {
			pc.l.Infof("error checking: %v", err)
		}
	case ppb.PlayerAction_PlayerActionFold:
		if err = pc.Fold(ctx); err != nil {
			pc.l.Infof("error folding: %v", err)
		}

	case ppb.PlayerAction_PlayerActionAllIn:
		// special case of bet for convenience
		// TODO: fix when money available
		// remains := pc.MyMoney().GetStack()
		// if err = pc.Bet(ctx, remains); err != nil {
		// 	pc.l.Infof("error betting: %v", err)
		// }
	case ppb.PlayerAction_PlayerActionBet:
		// TODO(sishi): under the gun has to raise at least a Big Blind if raising
		amount := paction.Opts.BetAmount
		if err = pc.Bet(ctx, amount); err != nil {
			pc.l.Infof("error betting: %v", err)
		}
	}

	// Send reply back to client
	pc.actionResult <- actions.NewPlayerActionResult(err == nil, err, nil)

	return nil
}

// Ack acks a token
func (pc *PokerClient) Ack(ctx context.Context, ackToken string) error {
	pc.l.Infof("Action: Ack [%v]", ackToken)

	req := &ppb.AckTokenRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		Token: ackToken,
	}

	_, err := pc.client.AckToken(ctx, req)
	if err != nil {
		pc.l.Fatal(err)
	} else {
		pc.l.Infof("Acked [%v]", ackToken)
		pc.lastAckedToken = ackToken
	}
	pc.l.Infof("acked: %v", ackToken)

	return nil
}

// Fold folds
func (pc *PokerClient) Fold(ctx context.Context) error {
	pc.l.Info("Action: Fold")

	action := ppb.PlayerAction_PlayerActionFold

	req := &ppb.TakeTurnRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		PlayerAction: action,
	}

	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// Check checks
func (pc *PokerClient) Check(ctx context.Context) error {
	pc.l.Info("Action: Check")

	action := ppb.PlayerAction_PlayerActionCheck

	req := &ppb.TakeTurnRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		PlayerAction: action,
	}
	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		pc.l.Error(err)
	}

	return nil
}

// Bet raises
func (pc *PokerClient) Bet(ctx context.Context, amount int64) error {
	pc.l.Infof("Action: Bet (%v)", amount)

	action := ppb.PlayerAction_PlayerActionBet

	req := &ppb.TakeTurnRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		PlayerAction: action,
		ActionOpts: &ppb.ActionOpts{
			BetAmount: amount,
		},
	}

	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// Call calls
func (pc *PokerClient) Call(ctx context.Context) error {
	pc.l.Info("Action: Call")

	action := ppb.PlayerAction_PlayerActionCall

	req := &ppb.TakeTurnRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		PlayerAction: action,
	}
	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		pc.l.Error(err)
	}

	return nil
}

// PrintHandResults prints the result
func (pc *PokerClient) PrintHandResults() error {

	fmt.Println("Results!!")
	// if !pc.roundFinished() {
	// 	return fmt.Errorf("hand not finished yet")
	// }

	// for _, p := range pc.TurnLog.Players() {
	// 	iswinner := ""
	// 	isme := ""

	// 	if pc.PlayerID == p.Id {
	// 		isme = "(me) "
	// 	}

	// 	for _, w := range pc.TurnLog.Winners() {
	// 		if p.Id == "" {
	// 			pc.l.Errorf(">w.ID:  %v", w.Id)
	// 			pc.l.Errorf(">p.ID:  %v", p.Id)
	// 			pc.l.Fatal("Missing p.ID from a player after game is over... is RoundInfo populated correctly?")
	// 		}
	// 		if p.Id == w.Id {
	// 			iswinner = "[winner] "
	// 		}
	// 	}

	// 	fmt.Printf("  %v%v%v ($%v) (%v)\n",
	// 		color.YellowString(isme),
	// 		color.GreenString(iswinner),
	// 		p.GetName(),
	// 		humanize.Comma(p.Money.GetWinnings()),
	// 		color.HiBlueString(p.Combo))
	// 	// show cards
	// 	pc.showCards(deck.CardsFromProto(p.GetHand()), false)
	// 	fmt.Println()
	// }

	return nil
}

// func (pc *PokerClient) showCards(cards []*deck.Card, divider bool) {

// 	if img, err := deck.CardsImage(cards, divider); err == nil {
// 		imgcat.CatImage(img, os.Stdout)
// 	}
// }

// Register registers with the server
func (pc *PokerClient) Register(ctx context.Context) error {

	pc.l.Info("Registering...")
	req := &ppb.RegisterRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
		},
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	var res *ppb.RegisterResponse
	var err error
	if res, err = pc.client.Register(ctx, req); err != nil {
		pc.l.Fatal(err)
	}
	pc.PlayerID = id.PlayerID(res.GetPlayerID())
	pc.l.Debugf("playerID: %v", pc.PlayerID)

	if pc.PlayerID == "" {
		return fmt.Errorf("Received blank playerID from server, but no error")
	}

	return nil
}

// JoinTable joins a new game
func (pc *PokerClient) JoinTable(ctx context.Context, wantTableID id.TableID) error {

	pc.l.Info("Joining table...")
	req := &ppb.JoinTableRequest{
		ClientInfo: &ppb.ClientInfo{
			PlayerName: pc.Name,
			PlayerID:   pc.PlayerID.String(),
			TableID:    pc.TableID.String(),
		},
		PlayerAction: ppb.PlayerAction_PlayerActionJoinTable,
	}

	var res *ppb.JoinTableResponse
	var err error
	if res, err = pc.client.JoinTable(ctx, req); err != nil {
		return err
	}
	tableID := id.TableID(res.GetTableID())
	pc.l.Debugf("tableID: %v", tableID)

	if tableID != wantTableID && wantTableID != "" {
		return fmt.Errorf("Asked to join table [%v], but joined [%v]", wantTableID, pc.TableID)
	}

	if tableID == "" {
		return fmt.Errorf("receieved empty table id from server, but no error")
	}

	if res.GetPosition() < 0 {
		return fmt.Errorf("received invalid position from server, but no error: %v", pc.position)
	}

	pc.position = res.GetPosition()
	pc.TableID = tableID

	return nil
}

// PlayHand plays the round
func (pc *PokerClient) PlayHand(ctx context.Context, handDone, doneLogStreaming chan bool) error {
	// var cachedState string

	// for !pc.roundFinished() {
	// 	s := pc.getGameState()
	// 	if s != cachedState {
	// 		pc.showGameState()
	// 		pc.showCards(append(pc.myCards(), deck.CardsFromProto(pc.TurnLog.CommunityCards().GetCard())...), true)
	// 		cachedState = s
	// 	}

	// 	if err := pc.TakeTurn(ctx); err != nil {
	// 		pc.l.Debugf("[%v] error taking turn: %v", pc.Name, err)
	// 	}

	// 	time.Sleep(time.Second * 1)
	// }

	// pc.l.Info("Hand finished, getting results...")
	// handDone <- true
	return nil
}

// func (pc *PokerClient) showGameState() {
// 	fmt.Println(pc.getGameState())
// }

// MyMoney returns the money for the current player
// func (pc *PokerClient) MyMoney() *ppb.PlayerMoney {

// 	for _, p := range pc.TurnLog.Players() {
// 		if p.Id == pc.PlayerID {
// 			return p.GetMoney()
// 		}
// 	}

// 	return nil
// }

// func (pc *PokerClient) getGameState() string {

// mycards := pc.myCards()
// mymoney := pc.MyMoney()

// var turnID int64 = -1
// var roundStatus = ppb.RoundStatus_RoundStatusInitializing
// var roundID string

// if len(pc.TurnLog.Current().GetTurns()) > 0 {
// 	turnID = pc.TurnLog.TurnID()
// 	roundStatus = pc.TurnLog.CurrentStatus()
// 	roundID = pc.TurnLog.LastRoundInfo().RoundID
// }

// var state strings.Builder

// state.WriteString(fmt.Sprintln("================================================================="))
// state.WriteString(fmt.Sprintf("%v (pos: %v) %v\n", color.GreenString("My Player:"), pc.position, pc.Name))
// state.WriteString(fmt.Sprintf("%v %v (myTurnID: %v)\n", color.GreenString("TurnID:"), turnID, pc.lastTurnTaken))
// state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Table State:"), pc.TableInfo.CurrentStatus().String()))

// if pc.TableInfo.GameStartsIn() > 0 {
// 	state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Game Starts In:"), pc.TableInfo.GameStartsIn().String()))
// }
// state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Round State:"), roundStatus))
// state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Round ID (local):"), pc.RoundID))
// state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Round ID (remote):"), roundID))

// if pc.TableInfo.CurrentStatus() >= ppb.TableStatus_TableStatusGamePlaying {
// 	state.WriteString(fmt.Sprintf("%v %v\n", color.RedString("Board:"), pc.protoToCards(pc.TurnLog.CommunityCards().GetCard())))
// 	state.WriteString(fmt.Sprintf("%v %v\n", color.RedString("Player Cards:"), mycards))

// 	if mymoney != nil {
// 		state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Total Stack:"), humanize.Comma(mymoney.GetStack())))
// 		state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Total Bet this Hand:"), humanize.Comma(mymoney.GetBetThisHand())))
// 		state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Current Bet:"), humanize.Comma(mymoney.GetBetThisRound())))
// 		state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Min Bet Right Now:"), humanize.Comma(mymoney.GetMinBetThisRound())))
// 		state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Pot Right Now:"), humanize.Comma(mymoney.GetPot())))
// 	}
// 	color.Unset()
// }

// state.WriteString(fmt.Sprintln(color.GreenString("All Players:")))
// for _, p := range pc.TurnLog.Players() {
// 	var me string
// 	if p.GetName() == pc.Name {
// 		me = color.HiGreenString("(me) ")
// 	}
// 	state.WriteString(fmt.Sprintf("  %v%v: %v\n", me, p.GetName(), p.GetState()))
// }
// state.WriteString(fmt.Sprintln("================================================================="))

// return state.String()
// }

// func (pc *PokerClient) myCards() []*deck.Card {
// 	return deck.CardsFromProto(pc.TurnLog.PlayerHole())
// }

func (pc *PokerClient) protoToCards(cards []*ppb.Card) []*deck.Card {
	nc := make([]*deck.Card, len(cards))

	for i := range cards {
		nc[i] = deck.NewCard(cards[i].Suite, cards[i].Rank)
	}

	return nc
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile(*grpcCrt)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
		// Does not do hostname verification
		InsecureSkipVerify: true,
	}

	return credentials.NewTLS(config), nil
}

func showWelcome() {
	// logo.PrintLogo()

	text := fmt.Sprintf("%v", game)
	myFigure := figure.NewColorFigure(text, "", "cyan", true)
	myFigure.Print()

	// showRandomCards(7)
}

func showRandomCards(num int) {
	rcards := make([]*deck.Card, num)
	for i := 0; i < len(rcards); i++ {
		rcards[i] = deck.RandomCard()
	}

	if img, err := deck.CardsImage(rcards, false); err == nil {
		imgcat.CatImage(img, os.Stdout)
	} else {
		log.Fatal(err)
	}

}
