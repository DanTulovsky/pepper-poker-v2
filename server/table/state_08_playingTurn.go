package table

import (
	"time"

	"github.com/DanTulovsky/deck"
)

type playingTurnState struct {
	baseState
}

func (i *playingTurnState) Init() error {
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
	i.l.Infof("Dealing the turn... [%v]", c)

	// next available player after the button goes first
	i.table.currentTurn = i.table.playerAfter(i.table.buttonPosition)

	pnt := i.table.positions[i.table.currentTurn]
	i.l.Infof("Player %s (%d) goes first", pnt.Name, i.table.currentTurn)
	pnt.WaitSince = time.Now()

	// records players that reached here
	for _, p := range i.table.CurrentHandActivePlayers() {
		p.Stats.StateInc("turn")
	}

	i.initrun = true
	return nil
}

func (i *playingTurnState) Tick() error {
	if !i.initrun {
		i.Init()
		return nil
	}

	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.haveWinner() {
		return i.table.setState(i.table.playingDoneState)
	}

	if i.table.canAdvanceState() {
		return i.table.setState(i.table.playingRiverState)
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
