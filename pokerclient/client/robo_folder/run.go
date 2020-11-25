// package main ...
// A very simple robot that folds every time
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/fatih/color"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/actions"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/roboclient"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	name            = flag.String("name", fmt.Sprintf("[robot_folder]-%v", randomdata.SillyName()), "player name")
	insecure        = flag.Bool("insecure", false, "if true, use insecure connection to server")
	roundStartDelay = flag.Duration("round_start_delay", time.Second, "delay between rounds")
	logg            *logger.Logger
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

	// Now play round until done
	for {
		logg.Info("Playing a game...")
		if err := r.PlayGame(ctx, cc); err != nil {
			logg.Error(err)
		}
		r.PokerClient.Reset()

		logg.Infof("Sleeping for %v", roundStartDelay)
		time.Sleep(*roundStartDelay)
	}
}

// decideOnAction decides what to do based on tableInfo state
func decideOnAction(pc *pokerclient.PokerClient) (*actions.PlayerAction, error) {
	paction := ppb.PlayerAction_PlayerActionFold

	// First three fields are not used and are set automatically by the client
	playerAction := actions.NewPlayerAction(id.EmptyPlayerID, id.EmptyTableID, paction, nil, nil)

	logg.Infof("Taking action: %v", paction.String())
	return playerAction, nil
}
