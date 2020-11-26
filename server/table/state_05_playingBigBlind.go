package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingBigBlindState struct {
	baseState
}

func (i *playingBigBlindState) Init() error {
	bigBlind := i.table.positions[i.table.currentTurn]
	i.l.Infof("[%v] putting in big blind...", bigBlind.Name)

	i.table.advancePlayer()
	return nil
}

func (i *playingBigBlindState) Bet(p *player.Player, bet int64) error {
	return fmt.Errorf("only big blind bets")
}

func (i *playingBigBlindState) Call(p *player.Player) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingBigBlindState) Check(p *player.Player) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingBigBlindState) Fold(p *player.Player) error {
	return fmt.Errorf("cannot fold during this round")
}

func (i *playingBigBlindState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	// // Put money into pot
	// if i.round.players[i.round.CurrentTurn()] == i.round.bigBlindPlayer {
	// 	bet := i.round.bigBlind
	// 	if i.round.bigBlindPlayer.Stack() < bet {
	// 		bet = i.round.bigBlindPlayer.Stack()
	// 	}
	// 	if err := i.round.bet(i.round.bigBlindPlayer.ID(), bet); err != nil {
	// 		log.Fatalf("playingBigBlindState error: %s", err)
	// 	}
	// } else {
	// 	log.Fatal("playingBigBlindState error, should never happen.")
	// }

	// // go to next statess
	// i.round.advancePlayer()
	// i.round.bigBlindPlayer.actionRequiredThisRound = !i.round.bigBlindPlayer.allin
	// i.round.setState(i.round.roundPreFlop, true, false)
	i.table.setState(i.table.playingPreFlopState)

	return nil
}

func (i *playingBigBlindState) WaitingTurnPlayer() *player.Player {
	return nil
}
