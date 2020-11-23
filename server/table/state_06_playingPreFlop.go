package table

import "github.com/DanTulovsky/pepper-poker-v2/server/player"

type playingPreFlopState struct {
	baseState
}

func (i *playingPreFlopState) bet(p *player.Player, bet int64) error {
	return i.table.bet(p, bet)
}

func (i *playingPreFlopState) call(p *player.Player) error {
	return i.table.call(p)
}

func (i *playingPreFlopState) check(p *player.Player) error {
	return i.table.check(p)
}

func (i *playingPreFlopState) fold(p *player.Player) error {
	return i.table.fold(p)
}

func (i *playingPreFlopState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	// if i.round.haveWinner() {
	// 	i.round.currentTurn = -1
	// 	i.round.setState(i.round.roundDone, true, false)
	// 	return nil
	// }

	// // Move to next state once all players took their turn
	// if i.round.canAdvanceState() {
	// 	return i.round.advanceStateAndReset(i.round.roundFlop, true)
	// }

	// // Advance active player
	// if !i.round.players[i.round.currentTurn].actionRequiredThisRound {
	// 	i.round.advancePlayer()
	// }

	i.table.setState(i.table.playingFlopState)
	return nil
}
