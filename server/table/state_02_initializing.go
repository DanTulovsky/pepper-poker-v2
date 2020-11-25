package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/acks"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type initializingState struct {
	baseState

	token *acks.Token
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

	// reset any existing acks
	i.table.clearAckToken()

	// Used to get an ack before game starts
	i.token = acks.New(i.table.ActivePlayers(), i.table.defaultAckTimeout)
	i.token.StartTime()
	i.table.setAckToken(i.token)
}

func (i *initializingState) StartGame() error {
	return fmt.Errorf("game not ready to start yet")
}

func (i *initializingState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.token.AllAcked() {
		i.table.clearAckToken()
		i.token = nil
		i.table.setState(i.table.readyToStartState)
		return nil
	}

	// token expired, we don't have all acks
	if i.token.Expired() {
		i.l.Infof("some [%d] players failed to ack, resetting", i.token.NumStillToAck())

		// TODO: Kick out only those players that failed to ack and then go back to beginning state
		i.table.reset()
		return nil
	}

	i.l.Infof("Waiting (%v) for %d players to ack...", i.token.TimeRemaining().Truncate(time.Second), i.token.NumStillToAck())

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

func (i *initializingState) Reset() {
	i.token = nil
}
