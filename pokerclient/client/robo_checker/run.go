// package main ...
// A very simple robot that folds every time
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/fatih/color"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/actions"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/roboclient"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	name     = flag.String("name", fmt.Sprintf("[robot_checker]-%v", randomdata.SillyName()), "player name")
	insecure = flag.Bool("insecure", false, "if true, use insecure connection to server")
	logg     *logger.Logger
)

const (
	game    string = "Pepper-Poker"
	version string = "0.1-pre-alpha"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	logg = logger.New("main", color.New(color.FgCyan))

	logg.Info("Starting client...")

	ctx := context.Background()
	cc := roboclient.NewCommChannels()

	r, err := roboclient.NewRoboClient(ctx, *name, decideOnAction, cc, *insecure)
	if err != nil {
		logg.Fatal(err)
	}

	if err := r.JoinGame(ctx); err != nil {
		logg.Fatal(err)
	}

	// Now play game until quit
	logg.Info("Playing a game...")
	donec := make(chan bool)
	SetupCloseHandler(donec)

	if err := r.PlayGame(ctx, cc, donec); err != nil {
		logg.Error(err)
	}

}

func decideOnAction(data *ppb.GameData) (*actions.PlayerAction, error) {
	logg.Info("Deciding on action...")

	// This client either calls if required, checks, or goes all in
	var paction ppb.PlayerAction

	mymoney := data.GetPlayer().GetMoney()
	switch {
	case mymoney.BetThisRound < mymoney.MinBetThisRound:
		switch {
		case mymoney.Stack > mymoney.MinBetThisRound-mymoney.BetThisRound:
			// Have enough to call
			paction = ppb.PlayerAction_PlayerActionCall
		default:
			// Must go All In (we never fold!)
			paction = ppb.PlayerAction_PlayerActionAllIn
		}
	default:
		// Check
		paction = ppb.PlayerAction_PlayerActionCheck
	}

	// First three fields are not used and are set automatically by the client
	playerAction := actions.NewPlayerAction(id.EmptyPlayerID, id.EmptyTableID, paction, nil, nil)

	logg.Infof("Taking action: %v", paction.String())
	return playerAction, nil
}

// SetupCloseHandler creates a 'listener' on a new goroutine which will notify the
// program if it receives an interrupt from the OS. We then handle this by calling
// our clean up procedure and exiting the program.
func SetupCloseHandler(donec chan bool) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		donec <- true
		// os.Exit(0)
	}()
}