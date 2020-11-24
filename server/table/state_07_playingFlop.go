package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingFlopState struct {
	baseState
}

func (i *playingFlopState) bet(p *player.Player, bet int64) error {
	return i.table.bet(p, bet)
}

func (i *playingFlopState) call(p *player.Player) error {
	return i.table.call(p)
}

func (i *playingFlopState) check(p *player.Player) error {
	return i.table.check(p)
}

func (i *playingFlopState) fold(p *player.Player) error {
	return i.table.fold(p)
}

func (i *playingFlopState) Init() {
	i.table.SetPlayersActionRequired()
	i.l.Info("Dealing the flop...")
}

func (i *playingFlopState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	// if !i.dealt {
	// 	i.logger.Info("Dealing the Flop...")
	// 	// Burn one.
	// 	if _, err := i.round.deck.Next(); err != nil {
	// 		return err
	// 	}
	// 	// Deal the flop.
	// 	for j := 0; j < 3; j++ {
	// 		var c *deck.Card
	// 		var err error
	// 		if c, err = i.round.deck.Next(); err != nil {
	// 			return err
	// 		}
	// 		i.round.board.cards = append(i.round.board.cards, c)
	// 	}
	// 	i.dealt = true

	// 	i.round.LogSystemTurn(i.round.tableID, i.round.id, ppb.Action_ActionDealCard, &ppb.ActionOpts{
	// 		Card: deck.CardsToProto(i.round.board.cards),
	// 	})
	// }

	// if i.round.haveWinner() {
	// 	i.round.currentTurn = -1
	// 	i.round.setState(i.round.roundDone, true, false)
	// 	return nil
	// }

	// // Move to next state once all players took their turn
	// if i.round.canAdvanceState() {
	// 	return i.round.advanceStateAndReset(i.round.roundTurn, true)
	// }

	// // Advance active player
	// if !i.round.players[i.round.currentTurn].actionRequiredThisRound {
	// 	i.round.advancePlayer()
	// }

	i.table.setState(i.table.playingTurnState)
	return nil
}
