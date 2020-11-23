package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type gameReadyToStartState struct {
	baseState
	delay         time.Duration
	playerTimeout time.Duration
}

func (i *gameReadyToStartState) StartGame() error {
	i.l.Info("Starting new game with players...")

	i.l.Info("Dealings cards to players...")
	// i.table.button = i.table.PlayerAfter(i.table.button)

	// activePlayers := []*poker.PlayerInfo{}

	// for _, p := range i.table.AvailablePlayers() {
	// 	// Assume the player buys in with all the money they have for now
	// 	i.table.BuyIn(p, p.bank)
	// 	pi := poker.NewPlayerInfo(p.ID(), p.Name(), p.Stack())

	// 	p.stack = 0 // delegate the stack to the round
	// 	activePlayers = append(activePlayers, pi)
	// }

	// for _, p := range activePlayers {
	// 	i.l.Infof("  [%v ($%v)]: %v", p.Name(), humanize.Comma(p.Stack()), p.Hole())
	// }
	// var err error
	// i.table.round, err = poker.NewRound(i.table.id, activePlayers, i.table.smallBlind, i.table.bigBlind, i.delay, i.playerTimeout)
	// if err != nil {
	// 	return err
	// }
	// i.l.Infof("Started new round: %v", i.table.Round().ID())

	return nil
}

func (i *gameReadyToStartState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if err := i.StartGame(); err != nil {
		return err
	}

	// i.l.Infof("Table [%v] starting round... (players: %d)", i.table.id, len(i.table.Round().Players()))
	// for _, p := range i.table.Round().Players() {
	// 	i.l.Infof("  %v", p)
	// }

	i.table.setState(i.table.playingSmallBlindState)
	return nil
}

func (i *gameReadyToStartState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already started, wait for next round")
}

// WhoseTurn returns the player whose turn it is.
func (i *gameReadyToStartState) WhoseTurn() *player.Player {
	return nil
}
