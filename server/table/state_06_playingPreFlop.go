package table

import "time"

type playingPreFlopState struct {
	baseState
}

func (i *playingPreFlopState) Init() error {
	i.baseState.Init()

	i.table.SetPlayersActionRequired()

	// properly set from the previous state
	p := i.table.positions[i.table.currentTurn]

	i.l.Infof("Player %s (%d) goes first", p.Name, i.table.currentTurn)
	p.WaitSince = time.Now()

	// records players that reached here
	for _, p := range i.table.CurrentHandActivePlayers() {
		p.Stats.StateInc("preflop")
	}

	i.initrun = true
	return nil
}

func (i *playingPreFlopState) Tick() error {
	if !i.initrun {
		i.Init()
		return nil
	}

	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.haveWinner() {
		return i.table.setState(i.table.playingDoneState)
	}

	if i.table.canAdvanceState() {
		return i.table.setState(i.table.playingFlopState)
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
