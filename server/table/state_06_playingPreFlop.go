package table

type playingPreFlopState struct {
	baseState
}

func (i *playingPreFlopState) Init() error {
	i.table.SetPlayersActionRequired()

	// properly set from the previous state
	current := i.table.positions[i.table.currentTurn]

	i.l.Infof("Player %s (%d) goes first", current.Name, i.table.currentTurn)
	return nil
}

func (i *playingPreFlopState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.canAdvanceState() {
		i.table.setState(i.table.playingFlopState)
		return nil
	}

	current := i.table.positions[i.table.currentTurn]
	if !current.ActionRequired() {
		i.table.advancePlayer()
		return nil
	}

	// if i.round.haveWinner() {
	// 	i.round.currentTurn = -1
	// 	i.round.setState(i.round.roundDone, true, false)
	// 	return nil
	// }

	return nil
}
