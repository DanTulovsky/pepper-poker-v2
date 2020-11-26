package table

type playingRiverState struct {
	baseState
}

func (i *playingRiverState) Init() error {
	i.table.SetPlayersActionRequired()
	i.l.Info("Dealing the river...")

	// next available player after the button goes first
	i.table.currentTurn = i.table.playerAfter(i.table.button)

	current := i.table.positions[i.table.currentTurn]
	i.l.Infof("Player %s (%d) goes first", current.Name, i.table.currentTurn)

	return nil
}

func (i *playingRiverState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	if i.table.canAdvanceState() {
		i.table.setState(i.table.playingDoneState)
		return nil
	}

	current := i.table.positions[i.table.currentTurn]
	if !current.ActionRequired() {
		i.table.advancePlayer()
		return nil
	}

	// if !i.dealt {
	// 	i.logger.Info("Dealing the River...")
	// 	// Burn one.
	// 	c, error := i.round.deck.Next()
	// 	if error != nil {
	// 		return error
	// 	}
	// 	// Deal the river.
	// 	c, error = i.round.deck.Next()
	// 	if error != nil {
	// 		return error
	// 	}
	// 	i.round.board.cards = append(i.round.board.cards, c)
	// 	i.dealt = true

	// 	i.round.LogSystemTurn(i.round.tableID, i.round.id, ppb.Action_ActionDealCard, &ppb.ActionOpts{
	// 		Card: []*ppb.Card{c.ToProto()},
	// 	})
	// }

	// if i.round.haveWinner() {
	// 	i.round.currentTurn = -1
	// 	i.round.setState(i.round.roundDone, true, false)
	// 	return nil
	// }

	// // Move to next state once all players have acted
	// if i.round.canAdvanceState() {
	// 	return i.round.advanceStateAndReset(i.round.roundDone, true)
	// }

	// // Advance active player
	// if !i.round.players[i.round.currentTurn].actionRequiredThisRound {
	// 	i.round.advancePlayer()
	// }

	return nil
}
