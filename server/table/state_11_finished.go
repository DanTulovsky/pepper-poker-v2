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

func (i *finishedState) Bet(p *player.Player, bet int64) error {
	return fmt.Errorf("hand is done")
}

func (i *finishedState) Call(p *player.Player) error {
	return fmt.Errorf("hand is done")
}

func (i *finishedState) Check(p *player.Player) error {
	return fmt.Errorf("hand is done")
}

func (i *finishedState) Fold(p *player.Player) error {
	return fmt.Errorf("hand is done")
}

func (i *finishedState) Init() {
	i.start = time.Now()

}

func (i *finishedState) StartGame() error {
	return fmt.Errorf("game [%v] already finished", i.table.ID)
}

func (i *finishedState) Tick() error {

	delay := i.gameEndDelay - time.Now().Sub(i.start)
	i.l.Infof("Waiting %v before starting new game...", delay.Truncate(time.Second))

	if delay <= 0 {
		i.table.setState(i.table.waitingPlayersState)
	}

	return nil
}

func (i *finishedState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already finished, wait for next hand")
}

func (i *finishedState) WaitingTurnPlayer() *player.Player {
	return nil
}
