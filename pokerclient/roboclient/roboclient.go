package roboclient

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/fatih/color"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/actions"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// DeciderFunc is the function that decides what to do
type DeciderFunc func(data *ppb.GameData) (*actions.PlayerAction, error)

// RoboClient is a robot playing poker
type RoboClient struct {
	Name string

	playerID  id.PlayerID
	tableID   id.TableID
	gameState ppb.GameState
	gameData  *ppb.GameData

	PokerClient *pokerclient.PokerClient
	DeciderFunc DeciderFunc

	l *logger.Logger
}

// NewRoboClient returns a new robot client
func NewRoboClient(ctx context.Context, name string, df DeciderFunc, cc *CommChannels, insecure bool) (*RoboClient, error) {
	rand.Seed(time.Now().UnixNano())

	r := &RoboClient{
		Name:        name,
		DeciderFunc: df,
		l:           logger.New("roboclient", color.New(color.FgBlue)),
	}

	var err error
	if r.PokerClient, err = pokerclient.New(ctx, r.Name, insecure, cc.Paction, cc.Presult, cc.InputWanted); err != nil {
		return nil, err
	}
	return r, nil
}

// JoinGame says hello and joins table
func (r *RoboClient) JoinGame(ctx context.Context) error {

	// First register the name with the server
	r.l.Info("Registering with the server...")
	if err := r.PokerClient.Register(ctx); err != nil {
		return fmt.Errorf("failed to register with server: %v", err)
	}

	// Second join the table
	var err error
	if err = r.PokerClient.JoinTable(ctx, id.TableID("")); err != nil {
		return fmt.Errorf("failed to join game: %v", err)
	}

	r.l.Infof("[%v] Joined game [%v]...", r.PokerClient.Name, r.PokerClient.TableID)

	return nil
}

// PlayGame plays the game
func (r *RoboClient) PlayGame(ctx context.Context, cc *CommChannels, donegamec chan bool) error {

	errc := make(chan error) // used to get error from pokerclient thread
	donec := make(chan bool) // used to cancel background server receiever thread

	go r.PokerClient.Play(ctx, donec, errc)

	// Play until donec channel receives data
OUTER:
	for {
		select {
		case <-donegamec: // allows client to exit, sent from main
			r.l.Info("Game done...")
			break OUTER
		case err := <-errc:
			if err != nil {
				r.l.Error(err)
			}
			break OUTER
		case r.gameData = <-cc.InputWanted:
			if err := r.takeTurn(cc.Paction, cc.Presult); err != nil {
				r.l.Error(err)
			}
		default:
		}
	}

	// stop Info streaming
	r.l.Info("Telling info thread to shut down")
	select {
	case donec <- true:
	default:
	}

	return nil
}

func (r *RoboClient) takeTurn(paction chan *actions.PlayerAction, presult chan *actions.PlayerActionResult) error {
	r.l.Info("Taking turn...")

	var playerAction *actions.PlayerAction
	var err error

	for playerAction == nil {
		// action is sent over paction
		playerAction, err = r.DeciderFunc(r.gameData)
		if err != nil {
			r.l.Error(err)
		}
	}
	paction <- playerAction

	// result comes back as presult

	// block until we get a result
	r.l.Info("Waiting for result")
	result := <-presult
	if !result.Success() {
		return result.Error()
	}

	r.l.Info("Got result")
	return nil
}

// CommChannels encapsulate the comm channels to pokerClient
type CommChannels struct {
	Paction     chan *actions.PlayerAction
	Presult     chan *actions.PlayerActionResult
	InputWanted chan *ppb.GameData
}

// NewCommChannels returns a new commchannels
func NewCommChannels() *CommChannels {

	return &CommChannels{
		// Communication with pokerClient
		Paction: make(chan *actions.PlayerAction),
		Presult: make(chan *actions.PlayerActionResult),
		// When the client needs input, it sends a message on this channel with the current TableInfo proto
		InputWanted: make(chan *ppb.GameData),
	}
}
