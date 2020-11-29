package table

type playingPreFlopState struct {
	baseState
}

func (i *playingPreFlopState) Init() error {
	i.baseState.Init()

	i.table.SetPlayersActionRequired()

	// properly set from the previous state
	current := i.table.positions[i.table.currentTurn]

	i.l.Infof("Player %s (%d) goes first", current.Name, i.table.currentTurn)
	return nil
}

func (i *playingPreFlopState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.haveWinner() {
		i.table.setState(i.table.playingDoneState)
	}

	if i.table.canAdvanceState() {
		i.table.setState(i.table.playingFlopState)
		return nil
	}

	current := i.table.positions[i.table.currentTurn]
	if !current.ActionRequired() {
		i.table.advancePlayer()
		return nil
	}

	return nil
}
