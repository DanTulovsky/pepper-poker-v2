package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingRiverState struct {
	baseState
}

func (i *playingRiverState) bet(p *player.Player, bet int64) error {
	return i.table.bet(p, bet)
}

func (i *playingRiverState) call(p *player.Player) error {
	return i.table.call(p)
}

func (i *playingRiverState) check(p *player.Player) error {
	return i.table.check(p)
}

func (i *playingRiverState) fold(p *player.Player) error {
	return i.table.fold(p)
}

func (i *playingRiverState) Init() {
	i.l.Info("Dealing the river...")
}

func (i *playingRiverState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

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

	i.table.setState(i.table.playingDoneState)
	return nil
}
