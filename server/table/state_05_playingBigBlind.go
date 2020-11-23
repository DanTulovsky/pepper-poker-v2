package table

import (
	"fmt"
)

type playingBigBlindState struct {
	baseState
}

func (i *playingBigBlindState) Bet(id string, bet int64) error {
	return fmt.Errorf("only big blind bets")
}

func (i *playingBigBlindState) Call(id string) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingBigBlindState) Check(id string) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingBigBlindState) Fold(id string) error {
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
