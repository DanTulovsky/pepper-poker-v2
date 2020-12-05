package table

import (
	"time"

	"github.com/DanTulovsky/deck"
)

type playingRiverState struct {
	baseState
}

func (i *playingRiverState) Init() error {
	i.baseState.Init()
	i.table.ResetPlayersBets()
	i.table.SetPlayersActionRequired()

	// Burn one.
	if _, err := i.table.deck.Next(); err != nil {
		return err
	}
	// Deal the flop.
	var c deck.Card
	var err error
	if c, err = i.table.deck.Next(); err != nil {
		return err
	}
	i.table.board.AddCard(c)
	i.l.Infof("Dealing the river... [%v]", c)

	// next available player after the button goes first
	i.table.currentTurn = i.table.playerAfter(i.table.button)

	p := i.table.positions[i.table.currentTurn]
	i.l.Infof("Player %s (%d) goes first", p.Name, i.table.currentTurn)
	p.WaitSince = time.Now()

	// records players that reached here
	for _, p := range i.table.CurrentHandActivePlayers() {
		p.Stats.StateInc("river")
	}
	return nil
}

func (i *playingRiverState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.haveWinner() {
		i.table.setState(i.table.playingDoneState)
	}

	if i.table.canAdvanceState() {
		i.table.setState(i.table.playingDoneState)
		return nil
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
