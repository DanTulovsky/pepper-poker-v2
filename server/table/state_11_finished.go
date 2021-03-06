package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/acks"
	"github.com/DanTulovsky/pepper-poker-v2/poker"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type finishedState struct {
	baseState

	gameEndDelay time.Duration
	gameEndTime  time.Time

	token       *acks.Token
	statusCache string
}

func (i *finishedState) Init() error {
	i.baseState.Init()

	// reset any existing acks
	i.table.clearAckToken()

	// Used to get an ack before game ends
	i.token = acks.New(i.table.CurrentHandPlayers(), i.table.defaultAckTimeout)
	i.token.StartTimer()
	i.table.setAckToken(i.token)

	i.gameEndTime = time.Now()

	i.initrun = true
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
	if !i.initrun {
		i.Init()
		return nil
	}

	now := time.Now()

	if i.token.NumStillToAck() > 0 && !i.token.Expired() {
		status := fmt.Sprintf("Waiting (%v) for %d players to ack...", i.token.TimeRemaining().Truncate(time.Second), i.token.NumStillToAck())
		if i.statusCache != status {
			i.l.Infof(status)
			i.statusCache = status
		}
	}

	if i.token.AllAcked() || i.token.Expired() {
		r := i.gameEndDelay - now.Sub(i.gameEndTime)

		status := fmt.Sprintf("Waiting (%v) before finishing game...", r.Truncate(time.Second))
		if i.statusCache != status {
			i.l.Infof(status)
			i.statusCache = status
		}

		if r < 0 {
			i.table.clearAckToken()
			i.token = nil

			i.l.Info("Removing players from current hand...")
			i.table.ClearCurrentHandPlayers()

			i.table.resetStates()
			// Stop sending old hand info to players... could be done better...
			i.table.board = poker.NewBoard()
			return i.table.setState(i.table.waitingPlayersState)
		}
	}

	return nil
}

func (i *finishedState) WaitingTurnPlayer() *player.Player {
	return nil
}
