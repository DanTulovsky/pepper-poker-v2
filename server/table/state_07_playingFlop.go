package table

import (
	"time"

	"github.com/DanTulovsky/deck"
)

type playingFlopState struct {
	baseState
}

func (i *playingFlopState) Init() error {
	i.baseState.Init()
	i.table.ResetPlayersBets()
	i.table.SetPlayersActionRequired()

	// Burn one.
	if _, err := i.table.deck.Next(); err != nil {
		return err
	}
	// Deal the flop.
	for j := 0; j < 3; j++ {
		var c deck.Card
		var err error
		if c, err = i.table.deck.Next(); err != nil {
			return err
		}
		i.table.board.AddCard(c)
	}
	i.l.Infof("Dealing the Flop... [%v]", i.table.board.Cards())

	// next available player after the button goes first
	i.table.currentTurn = i.table.playerAfter(i.table.buttonPosition)

	p := i.table.positions[i.table.currentTurn]
	i.l.Infof("Player %s (%d) goes first", p.Name, i.table.currentTurn)
	p.WaitSince = time.Now()

	// records players that reached here
	for _, p := range i.table.CurrentHandActivePlayers() {
		p.Stats.StateInc("flop")
	}
	return nil
}

func (i *playingFlopState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.haveWinner() {
		return i.table.setState(i.table.playingDoneState)
	}

	if i.table.canAdvanceState() {
		return i.table.setState(i.table.playingTurnState)
	}

	p := i.table.positions[i.table.currentTurn]
	if p == nil {
		return nil
	}
	i.table.FoldIfTurnTimerEnd(p)

	if !p.ActionRequired() {
		i.table.advancePlayer()
		return nil
	}

	return nil
}
