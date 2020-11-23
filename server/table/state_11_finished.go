package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type finishedState struct {
	baseState

	gameEndDelay time.Duration
	start        time.Time
}

func (i *finishedState) Init() {
	i.start = time.Now()

}

func (i *finishedState) StartGame() error {
	return fmt.Errorf("game [%v] already finished", i.table.ID)
}

func (i *finishedState) Tick() error {

	if time.Now().Sub(i.start) > i.gameEndDelay {
		i.table.setState(i.table.waitingPlayersState)
	}

	return nil
}

func (i *finishedState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already finished, wait for next round")
}

// WhoseTurn returns the player whose turn it is.
func (i *finishedState) WhoseTurn() *player.Player {
	return nil
}
