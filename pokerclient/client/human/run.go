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
	"strconv"
	"syscall"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/fatih/color"
	"github.com/tcnksm/go-input"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/actions"
	"github.com/DanTulovsky/pepper-poker-v2/pokerclient/roboclient"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	name     = flag.String("name", fmt.Sprintf("[robot_folder]-%v", randomdata.SillyName()), "player name")
	insecure = flag.Bool("insecure", false, "if true, use insecure connection to server")
	logg     *logger.Logger

	ui *input.UI = &input.UI{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}
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

	// Play
	for {
		query := "What to do?"
		action, err := ui.Select(query, []string{"play", "quit"}, &input.Options{
			Default:  "play",
			Loop:     true,
			Required: true,
		})
		if err != nil {
			logg.Fatal(err)
		}

		switch action {
		case "quit":
			os.Exit(0)
		default:
			// Now play game until quit
			logg.Info("Playing a game...")
			donec := make(chan bool)
			SetupCloseHandler(donec)

			if err := r.PlayGame(ctx, cc, donec); err != nil {
				logg.Error(err)
			}
		}
	}

}

// decideOnAction decides what to do based on tableInfo state
func decideOnAction(data *ppb.GameData) (*actions.PlayerAction, error) {
	query := "Select action..."

	action, err := ui.Select(query, []string{
		ppb.PlayerAction_PlayerActionFold.String(),
		ppb.PlayerAction_PlayerActionCheck.String(),
		ppb.PlayerAction_PlayerActionCall.String(),
		ppb.PlayerAction_PlayerActionAllIn.String(),
		ppb.PlayerAction_PlayerActionBet.String()}, &input.Options{
		Default:  ppb.PlayerAction_PlayerActionCheck.String(),
		Loop:     true,
		Required: true,
	})
	if err != nil {
		return nil, err
	}
	logg.Infof("selected: %v", action)

	paction := ppb.PlayerAction(ppb.PlayerAction_value[action])
	opts := &ppb.ActionOpts{}

	switch paction {
	case ppb.PlayerAction_PlayerActionBet:
		amount, err := betAmount(data)
		if err != nil {
			return nil, err
		}
		opts.BetAmount = amount
	default:
	}

	// First three fields are not used and are set automatically by the client
	playerAction := actions.NewPlayerAction(id.EmptyPlayerID, id.EmptyTableID, paction, opts, nil)

	return playerAction, err
}

// betAmount asks the user for and returns the amoount to bet
func betAmount(data *ppb.GameData) (int64, error) {
	query := "Select amount to bet"
	q, err := ui.Select(query, []string{"Big Blind", "3X Big Blind", "Custom"}, &input.Options{
		Default:  "3X Big Blind",
		Loop:     true,
		Required: true,
	})
	if err != nil {
		return 0, err
	}
	logg.Infof("selected: %v", q)

	switch q {
	case "Big Blind":
		return data.GetBigBlind(), nil
	case "3X Big Blind":
		return data.GetBigBlind() * 3, nil
	default:
		// fall through
	}

	query = "How much to bet?"
	amountS, err := ui.Ask(query, &input.Options{
		Default:      "1",
		Loop:         true,
		Required:     true,
		ValidateFunc: validateAmount,
	})
	if err != nil {
		return 0, err
	}
	// ignore error since we validate above
	amount, err := strconv.ParseInt(amountS, 10, 64)
	return amount, err
}

// validateAmount is used by ui.Ask. Validates that the amount entered is a valid number.
func validateAmount(s string) error {
	_, err := strconv.Atoi(s)
	return err
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
