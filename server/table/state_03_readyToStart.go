package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/dustin/go-humanize"
)

type readyToStartState struct {
	baseState
	playerTimeout time.Duration
}

func (i *readyToStartState) Init() error {
	i.l.Info("Starting new game with players...")

	i.l.Info("Players buying in...")
	for _, p := range i.table.ActivePlayers() {
		// TODO: Assume the player buys in with all the money they have for now
		i.table.BuyIn(p, p.Money().Bank())

		i.l.Infof("  [%v ($%v)]: %v", p.Name, humanize.Comma(p.Money().Stack()), p.Hole())
	}

	i.l.Info("Dealings cards to players...")
	for j := 0; j < 2; j++ {
		for _, p := range i.table.ActivePlayers() {
			card, err := i.table.deck.Next()
			if err != nil {
				return err
			}

			p.AddHoleCard(card)
		}
	}

	return nil
}

func (i *readyToStartState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	i.table.setState(i.table.playingSmallBlindState)
	return nil
}

func (i *readyToStartState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already started, wait for next round")
}

// WhoseTurn returns the player whose turn it is.
func (i *readyToStartState) WhoseTurn() *player.Player {
	return nil
}

func (i *readyToStartState) WaitingTurnPlayer() *player.Player {
	return nil
}
