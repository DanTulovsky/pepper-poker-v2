package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/acks"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type initializingState struct {
	baseState

	token       *acks.Token
	statusCache string
}

func (i *initializingState) Init() error {
	i.l.Info("Initializing table...")

	i.table.currentHand++

	i.table.buttonPosition = i.table.playerAfter(i.table.buttonPosition)
	i.table.smallBlindPosition = i.table.playerAfter(i.table.buttonPosition)
	i.table.bigBlindPosition = i.table.playerAfter(i.table.smallBlindPosition)

	i.table.smallBlindPlayer = i.table.positions[i.table.smallBlindPosition]
	i.table.bigBlindPlayer = i.table.positions[i.table.bigBlindPosition]

	i.table.currentTurn = i.table.smallBlindPosition

	i.l.Infof("button: %v", i.table.positions[i.table.buttonPosition].Name)
	i.l.Infof("smallBlind: %v", i.table.smallBlindPlayer.Name)
	i.l.Infof("bigBlind: %v", i.table.bigBlindPlayer.Name)

	i.l.Info("Initializing player information for the hand...")
	for _, p := range i.table.CurrentHandPlayers() {
		p.Init()
	}

	// reset board, pot and deck
	i.table.ResetPlayersBets()

	// Used to get an ack before game starts
	i.token = acks.New(i.table.CurrentHandPlayers(), i.table.defaultAckTimeout)
	i.token.StartTimer()
	i.table.setAckToken(i.token)

	return nil
}

func (i *initializingState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.token.AllAcked() || i.token.NumStillToAck() == 0 {
		i.table.clearAckToken()
		i.token = nil
		return i.table.setState(i.table.readyToStartState)
	}

	// token expired, we don't have all acks
	if i.token.Expired() {
		failed := i.token.DidNotAckPlayers()
		i.l.Infof("[%d] players (%v) failed to ack, removing them", i.token.NumStillToAck(), i.token.DidNotAckPlayers())

		for _, p := range failed {
			i.l.Infof("removing disconnected player: %v", p.Name)
			i.table.removePlayer(p)
		}

		// TODO: Kick out only those players that failed to ack and then go back to beginning state
		// i.table.reset()

		i.table.setState(i.table.waitingPlayersState)
		return nil

	}

	status := fmt.Sprintf("Waiting (%v) for %d players to ack...", i.token.TimeRemaining().Truncate(time.Second), i.token.NumStillToAck())
	if i.statusCache != status {
		i.l.Infof(status)
		i.statusCache = status
	}

	return nil
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
