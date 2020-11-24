package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type initializingState struct {
	baseState
}

func (i *initializingState) StartGame() error {
	return fmt.Errorf("game not ready to start yet")
}

func (i *initializingState) Init() {
	i.l.Info("Initializing table...")
	i.table.button = i.table.playerAfter(i.table.button)
	i.table.currentTurn = i.table.playerAfter(i.table.button)

	i.l.Infof("button: %v", i.table.positions[i.table.button].Name)

	i.l.Info("Initializing player information for the hand...")
	for _, p := range i.table.ActivePlayers() {
		p.InitHand()
	}
}

func (i *initializingState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	i.table.setState(i.table.readyToStartState)

	return nil
}

func (i *initializingState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already started, wait for next round")
}

// WhoseTurn returns the player whose turn it is.
func (i *initializingState) WhoseTurn() *player.Player {
	return nil
}

func (i *initializingState) WaitingTurnPlayer() *player.Player {
	return nil
}
