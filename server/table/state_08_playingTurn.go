package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingTurnState struct {
	baseState
}

func (i *playingTurnState) bet(p *player.Player, bet int64) error {
	return i.table.bet(p, bet)
}

func (i *playingTurnState) call(p *player.Player) error {
	return i.table.call(p)
}

func (i *playingTurnState) check(p *player.Player) error {
	return i.table.check(p)
}

func (i *playingTurnState) fold(p *player.Player) error {
	return i.table.fold(p)
}

func (i *playingTurnState) Init() {
	i.l.Info("Dealing the turn...")
}

func (i *playingTurnState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	// if !i.dealt {
	// 	i.logger.Info("Dealing the Turn...")
	// 	// Burn one.
	// 	if _, err := i.round.deck.Next(); err != nil {
	// 		return err
	// 	}
	// 	// Deal the turn.
	// 	var c *deck.Card
	// 	var err error
	// 	if c, err = i.round.deck.Next(); err != nil {
	// 		return err
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
	// 	return i.round.advanceStateAndReset(i.round.roundRiver, true)
	// }

	// // Advance active player
	// if !i.round.players[i.round.currentTurn].actionRequiredThisRound {
	// 	i.round.advancePlayer()
	// }
	i.table.setState(i.table.playingRiverState)

	return nil
}
