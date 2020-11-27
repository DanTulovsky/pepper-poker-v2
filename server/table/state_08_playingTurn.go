package table

import "github.com/DanTulovsky/deck"

type playingTurnState struct {
	baseState
}

func (i *playingTurnState) Init() error {
	i.table.SetPlayersActionRequired()

	i.l.Info("Dealing the turn...")

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

	// next available player after the button goes first
	i.table.currentTurn = i.table.playerAfter(i.table.button)

	current := i.table.positions[i.table.currentTurn]
	i.l.Infof("Player %s (%d) goes first", current.Name, i.table.currentTurn)

	return nil
}

func (i *playingTurnState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.canAdvanceState() {
		i.table.setState(i.table.playingRiverState)
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
