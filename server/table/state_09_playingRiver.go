package table

import "github.com/DanTulovsky/deck"

type playingRiverState struct {
	baseState
}

func (i *playingRiverState) Init() error {
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

	current := i.table.positions[i.table.currentTurn]
	i.l.Infof("Player %s (%d) goes first", current.Name, i.table.currentTurn)

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

	current := i.table.positions[i.table.currentTurn]
	if !current.ActionRequired() {
		i.table.advancePlayer()
		return nil
	}

	return nil
}
