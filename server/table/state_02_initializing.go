package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/deck"
	"github.com/DanTulovsky/pepper-poker-v2/acks"
	"github.com/DanTulovsky/pepper-poker-v2/poker"
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

	i.table.button = i.table.playerAfter(i.table.button)
	sb := i.table.playerAfter(i.table.button)
	bb := i.table.playerAfter(sb)

	i.table.currentTurn = sb

	i.table.smallBlindPlayer = i.table.positions[sb]
	i.table.bigBlindPlayer = i.table.positions[bb]

	i.l.Infof("button: %v", i.table.positions[i.table.button].Name)
	i.l.Infof("smallBlind: %v", i.table.smallBlindPlayer.Name)
	i.l.Infof("bigBlind: %v", i.table.bigBlindPlayer.Name)

	i.l.Info("Initializing player information for the hand...")
	for _, p := range i.table.CurrentHandPlayers() {
		p.Init()
	}

	// reset board, pot and deck
	i.table.board = poker.NewBoard()
	i.table.deck = deck.NewShuffledDeck()
	i.table.pot = poker.NewPot()
	i.table.ResetPlayersBets()

	// reset any existing acks
	i.table.clearAckToken()

	// Used to get an ack before game starts
	i.token = acks.New(i.table.CurrentHandPlayers(), i.table.defaultAckTimeout)
	i.token.StartTime()
	i.table.setAckToken(i.token)

	return nil
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
		i.l.Infof("some [%d] players failed to ack, resetting: %v", i.token.NumStillToAck(), i.token.DidNotAckPlayers())

		// TODO: Kick out only those players that failed to ack and then go back to beginning state
		i.table.reset()
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
