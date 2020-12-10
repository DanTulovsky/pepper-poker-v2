package pokerclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/dustin/go-humanize"
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
	showCardImages     = flag.Bool("show_card_images", false, "set to true to display card images in terminal")

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
	PlayerID id.PlayerID

	PlayerUsername string
	PlayerPassword string

	TableID  id.TableID
	position int64
	client   ppb.PokerServerClient

	// The background GameData goroutine sends server updates on this channel
	datac chan *ppb.GameData

	// the last acked token
	lastAckedToken string
	lastTurnTaken  int64
	handFinished   bool

	gameState ppb.GameState
	money     *ppb.PlayerMoney

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
func New(ctx context.Context, username, password string, insecure bool, actions chan *actions.PlayerAction, actionResult chan *actions.PlayerActionResult, inputWanted chan *ppb.GameData) (*PokerClient, error) {
	showWelcome()

	rand.Seed(time.Now().UnixNano())

	if *httpPort == "" {
		port, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		*httpPort = fmt.Sprintf("%d", port)
	}

	logger := logger.New(username, color.New(color.FgGreen))
	pc := &PokerClient{
		PlayerUsername: username,
		PlayerPassword: password,
		l:              logger,
		action:         actions,
		actionResult:   actionResult,
		inputWanted:    inputWanted,
		lastTurnTaken:  -1, // server starts at 0
		datac:          make(chan *ppb.GameData),
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

// ClientInfo returns the filled into ClientInfo proto
func (pc *PokerClient) ClientInfo() *ppb.ClientInfo {
	return &ppb.ClientInfo{
		PlayerUsername: pc.PlayerUsername,
		Password:       pc.PlayerPassword,
		PlayerID:       pc.PlayerID.String(),
		TableID:        pc.TableID.String(),
	}
}

// Play is called after joining table to begin streaming GameData
func (pc *PokerClient) Play(ctx context.Context, donec chan bool, errc chan error) {
	pc.l.Info("Starting GameData streamer..")

	ctxCancel, cancel := context.WithCancel(ctx)
	pc.cancel = cancel

	// Subscribe to GameData from the server after joing table
	reqPlay := &ppb.PlayRequest{
		ClientInfo:   pc.ClientInfo(),
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	stream, err := pc.client.Play(ctxCancel, reqPlay)
	if err != nil {
		errc <- err
		return
	}

	exitc := make(chan bool)
	go pc.receiveGameData(stream, donec, exitc)
	if err := pc.processGameData(ctx, exitc); err != nil {
		errc <- err
	}
}

// processGameData receives GameData on the channel and acts on it
func (pc *PokerClient) processGameData(ctx context.Context, exitc chan bool) error {
	var err error
	// Receive GameData on datac channel and act on it
OUTER:
	for {
		// process server messages if any (on datac channel)
		select {
		case <-exitc:
			err = fmt.Errorf("processGameData exiting due to request")
			break OUTER
		case in := <-pc.datac:
			pc.l.Debug("received game data in main thread")

			if pc.PlayerID != id.PlayerID(in.PlayerID) {
				pc.l.Fatal("Mismatch in playerID; expected: %v; got: %v", pc.PlayerID, id.PlayerID(in.PlayerID))
			}
			if pc.TableID != id.TableID(in.GetInfo().GetTableID()) {
				pc.l.Fatalf("Mismatch in tableID; expected: %v; got: %v", pc.TableID, id.TableID(in.GetInfo().GetTableID()))
			}
			pc.position = in.GetPlayer().GetPosition()
			pc.money = in.GetPlayer().GetMoney()

			waitName := in.WaitTurnName
			waitNum := in.WaitTurnNum
			waitTimeLeft := time.Duration(in.WaitTurnTimeLeftSec * 1000000000)

			pc.gameState = in.GetInfo().GetGameState()
			ackToken := in.GetInfo().GetAckToken()

			pc.l.Debugf("[tt: %v] Current Turn Player (num=%v): %v", waitTimeLeft, waitNum, waitName)
			pc.l.Debugf("Current State: %v", pc.gameState)

			switch in.GetPlayer().GetState() {
			case ppb.PlayerState_PlayerStateStackEmpty:
				if in.GetInfo().GetGameState() <= ppb.GameState_GameStateWaitingPlayers {
					if err = pc.BuyIn(ctx, in.GetInfo().GetBigBlind()); err != nil {
						pc.l.Infof("error buying in: %v", err)
						os.Exit(1)
					}
				}

			case ppb.PlayerState_PlayerStateBankEmpty:
				pc.l.Info("Ran out of money, exiting...")
				os.Exit(0)

			case ppb.PlayerState_PlayerStateCurrentTurn:

				if pc.lastTurnTaken < waitNum {
					pc.handFinished = false
					if err := pc.TakeTurn(ctx, in); err == nil {
						pc.lastTurnTaken = waitNum
					}
				}
			}

			if pc.handIsFinished() && !pc.handFinished {
				pc.l.Info("Game Finished!")

				pc.handFinished = true // used for display only
				pc.PrintHandResults(in)
			}

			pc.ackIfNeeded(ctx, ackToken)
		}
	}
	return err
}

func (pc *PokerClient) handIsFinished() bool {
	return pc.gameState == ppb.GameState_GameStatePlayingDone
}

// ackIfNeeded acks a token if needed
func (pc *PokerClient) ackIfNeeded(ctx context.Context, ackToken string) {

	if ackToken != pc.lastAckedToken && ackToken != "" {
		pc.l.Debugf("Acking [%v]", ackToken)
		pc.Ack(ctx, ackToken)
	}
}

// ReceiveGameData receives GameData from the server and sends it to the main thread over a channel
func (pc *PokerClient) receiveGameData(stream ppb.PokerServer_PlayClient, donec, exitc chan bool) error {
	pc.l.Debug("Started receive GameData thread...")

OUTER:
	for {
		select {
		case <-donec:
			pc.l.Info("calling cancel on server stream (stop called)")
			// cancel()
			// tell the processGameData loop to exit
			exitc <- true
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

	// Show most up to date status from the server
	pc.showGameState(in)

	pc.showCards(deck.CardsFromProto(append(in.GetPlayer().GetCard(), in.GetInfo().GetCommunityCards().Card...)), true)

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
		if err = pc.AllIn(ctx); err != nil {
			pc.l.Infof("error going all in: %v", err)
		}

	case ppb.PlayerAction_PlayerActionBet:
		// TODO(sishi): under the gun has to raise at least a Big Blind if raising
		amount := paction.Opts.BetAmount
		if err = pc.Bet(ctx, amount); err != nil {
			pc.l.Infof("error betting: %v", err)
		}
	}

	// Send reply back to client
	pc.actionResult <- actions.NewPlayerActionResult(err == nil, err, nil)

	return err
}

// Ack acks a token
func (pc *PokerClient) Ack(ctx context.Context, ackToken string) error {
	pc.l.Infof("Action: Ack [%v]", ackToken)

	req := &ppb.AckTokenRequest{
		ClientInfo: pc.ClientInfo(),
		Token:      ackToken,
	}

	_, err := pc.client.AckToken(ctx, req)
	if err != nil {
		pc.l.Fatal(err)
	}

	pc.l.Debugf("Acked [%v]", ackToken)
	pc.lastAckedToken = ackToken

	return nil
}

// Fold folds
func (pc *PokerClient) Fold(ctx context.Context) error {
	pc.l.Info("Action: Fold")

	action := ppb.PlayerAction_PlayerActionFold

	req := &ppb.TakeTurnRequest{
		ClientInfo:   pc.ClientInfo(),
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
		ClientInfo:   pc.ClientInfo(),
		PlayerAction: action,
	}
	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// Bet raises
func (pc *PokerClient) Bet(ctx context.Context, amount int64) error {
	pc.l.Infof("Action: Bet (%v)", amount)

	action := ppb.PlayerAction_PlayerActionBet

	req := &ppb.TakeTurnRequest{
		ClientInfo:   pc.ClientInfo(),
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

// AllIn goes all in
func (pc *PokerClient) AllIn(ctx context.Context) error {
	pc.l.Infof("Action: AllIn")

	action := ppb.PlayerAction_PlayerActionAllIn

	req := &ppb.TakeTurnRequest{
		ClientInfo:   pc.ClientInfo(),
		PlayerAction: action,
	}

	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// BuyIn sends the buyin request
func (pc *PokerClient) BuyIn(ctx context.Context, bigBlind int64) error {
	if pc.money.GetStack() > bigBlind {
		return nil
	}

	pc.l.Info("Action: BuyIn")

	action := ppb.PlayerAction_PlayerActionBuyIn

	req := &ppb.TakeTurnRequest{
		ClientInfo:   pc.ClientInfo(),
		PlayerAction: action,
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
		ClientInfo:   pc.ClientInfo(),
		PlayerAction: action,
	}
	_, err := pc.client.TakeTurn(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// IsWinner returns true if the players appears in the winners proto
func (pc *PokerClient) IsWinner(p *ppb.Player, winners []*ppb.Winners) bool {
	// TODO: Fix this
	for _, level := range winners {
		for _, w := range level.Ids {
			if p.Id == w {
				return true
			}
		}
	}
	return false
}

// PrintHandResults prints the result
func (pc *PokerClient) PrintHandResults(in *ppb.GameData) error {
	for _, p := range in.GetInfo().GetPlayers() {
		iswinner := ""
		isme := ""

		if pc.PlayerID.String() == p.Id {
			isme = "(me) "
		}

		if pc.IsWinner(p, in.GetInfo().GetWinningIds()) {
			iswinner = "[winner] "
		}

		fmt.Printf("  %v%v%v (+$%v; bank: %v; stack: %v) (%v)\n",
			color.YellowString(isme),
			color.GreenString(iswinner),
			p.GetName(),
			humanize.Comma(p.GetMoney().GetWinnings()),
			humanize.Comma(p.GetMoney().GetBank()),
			humanize.Comma(p.GetMoney().GetStack()),
			color.HiBlueString(p.GetCombo()))

		fmt.Printf("     [%v] %v", p.GetCombo(), deck.CardsFromProto(p.GetHand()))
		pc.showCards(deck.CardsFromProto(p.GetHand()), false)
		fmt.Println()
	}

	return nil
}

func (pc *PokerClient) showCards(cards []deck.Card, divider bool) {

	if !*showCardImages {
		return
	}

	if len(cards) == 0 {
		return
	}

	var img image.Image
	var err error

	if img, err = deck.CardsImage(cards, divider); err != nil {
		log.Fatal(err)
	}
	imgcat.CatImage(img, os.Stdout)
}

// Register registers with the server
func (pc *PokerClient) Register(ctx context.Context) error {

	pc.l.Info("Registering...")
	req := &ppb.RegisterRequest{
		ClientInfo:   pc.ClientInfo(),
		PlayerAction: ppb.PlayerAction_PlayerActionRegister,
	}
	var res *ppb.RegisterResponse
	var err error
	if res, err = pc.client.Register(ctx, req); err != nil {
		return err
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
		ClientInfo:   pc.ClientInfo(),
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
		pc.l.Info("Joined table, but do not have position yet until next hand.")
	}

	pc.position = res.GetPosition()
	pc.TableID = tableID

	return nil
}

func (pc *PokerClient) showGameState(in *ppb.GameData) {
	fmt.Println(pc.getGameState(in))
}

func (pc *PokerClient) getGameState(in *ppb.GameData) string {

	mycards := in.GetPlayer().GetCard()
	mymoney := in.GetPlayer().GetMoney()
	gameState := in.GetInfo().GetGameState()
	gameStartsIn := in.GetInfo().GetGameStartsInSec()
	buyin := in.GetInfo().GetBuyin()

	waitName := in.WaitTurnName
	waitTimeLeft := time.Duration(in.WaitTurnTimeLeftSec * 1000000000)

	var state strings.Builder

	state.WriteString(fmt.Sprintln("================================================================="))
	state.WriteString(fmt.Sprintf("%v (pos: %v) %v\n", color.GreenString("My Player:"), pc.position, pc.PlayerUsername))
	state.WriteString(fmt.Sprintf("%v %v (%v)\n", color.GreenString("Turn:"), waitName, waitTimeLeft))
	state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Table State:"), in.GetInfo().GetGameState()))
	state.WriteString(fmt.Sprintf("%v $%v\n", color.YellowString("Table Buyin:"), humanize.Comma(buyin)))

	startsIn := time.Duration(time.Second * time.Duration(gameStartsIn*1000000))
	if startsIn > 0 {
		state.WriteString(fmt.Sprintf("%v %v\n", color.YellowString("Game Starts In:"), startsIn.Truncate(time.Second)))
	}

	if gameState >= ppb.GameState_GameStatePlayingSmallBlind {
		state.WriteString(fmt.Sprintf("%v %v\n", color.RedString("Board:"), pc.protoToCards(in.GetInfo().GetCommunityCards().GetCard())))
		state.WriteString(fmt.Sprintf("%v %v\n", color.RedString("Player Cards:"), deck.CardsFromProto(mycards)))

		if mymoney != nil {
			state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Total Bank:"), humanize.Comma(mymoney.GetBank())))
			state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Total Stack:"), humanize.Comma(mymoney.GetStack())))
			state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Bet this Hand:"), humanize.Comma(mymoney.GetBetThisHand())))
			state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Bet this Betting Round:"), humanize.Comma(mymoney.GetBetThisRound())))
			state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Min Bet Right Now:"), humanize.Comma(mymoney.GetMinBetThisRound())))
			state.WriteString(fmt.Sprintf("%v $%v\n", color.CyanString("Pot Right Now:"), humanize.Comma(mymoney.GetPot())))
		}
		color.Unset()
	}

	state.WriteString(fmt.Sprintln(color.GreenString("All Players:")))
	for _, p := range in.GetInfo().GetPlayers() {
		var me string
		if p.GetId() == pc.PlayerID.String() {
			me = color.HiGreenString("(me) ")
		}
		state.WriteString(fmt.Sprintf("  %v%v (last_action: %v ($%v))\n", me, p.GetName(), p.GetLastAction().GetAction(), p.GetLastAction().GetAmount()))
	}
	state.WriteString(fmt.Sprintln("================================================================="))

	return state.String()
}

func (pc *PokerClient) protoToCards(cards []*ppb.Card) []deck.Card {
	nc := make([]deck.Card, len(cards))

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
	rcards := make([]deck.Card, num)
	for i := 0; i < len(rcards); i++ {
		rcards[i] = deck.RandomCard()
	}

	if img, err := deck.CardsImage(rcards, false); err == nil {
		imgcat.CatImage(img, os.Stdout)
	} else {
		log.Fatal(err)
	}

}
