package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/acks"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type finishedState struct {
	baseState

	gameEndDelay time.Duration

	token *acks.Token
}

func (i *finishedState) Init() error {

	// reset any existing acks
	i.table.clearAckToken()

	// Used to get an ack before game ends
	i.token = acks.New(i.table.ActivePlayers(), i.table.defaultAckTimeout)
	i.token.StartTime()
	i.table.setAckToken(i.token)

	return nil
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

func (i *finishedState) StartGame() error {
	return fmt.Errorf("game [%v] already finished", i.table.ID)
}

func (i *finishedState) Tick() error {

	i.l.Infof("Waiting (%v) for %d players to ack...", i.token.TimeRemaining().Truncate(time.Second), i.token.NumStillToAck())
	if i.token.AllAcked() || i.token.Expired() {
		i.table.clearAckToken()
		i.token = nil
		i.table.setState(i.table.waitingPlayersState)
		return nil
	}

	return nil
}

func (i *finishedState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already finished, wait for next hand")
}

func (i *finishedState) WaitingTurnPlayer() *player.Player {
	return nil
}
