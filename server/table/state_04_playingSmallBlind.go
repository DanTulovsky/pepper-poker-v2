package table

import (
	"fmt"
)

type playingSmallBlindState struct {
	baseState
}

func (i *playingSmallBlindState) Bet(id string, bet int64) error {
	return fmt.Errorf("only small blind bets")
}

func (i *playingSmallBlindState) Call(id string) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingSmallBlindState) Check(id string) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingSmallBlindState) Fold(id string) error {
	return fmt.Errorf("cannot fold during this round")
}

func (i *playingSmallBlindState) Init() {
	smallBlind := i.table.positions[i.table.currentTurn]
	i.l.Infof("[%v] putting in small blind...", smallBlind.Name)

	i.table.advancePlayer()
}

func (i *playingSmallBlindState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	// // Put money into pot
	// if i.round.players[i.round.CurrentTurn()] == i.round.smallBlindPlayer {
	// 	i.logger.Info("Putting money in for small blind")
	// 	bet := i.round.smallBlind
	// 	if i.round.smallBlindPlayer.Stack() < bet {
	// 		bet = i.round.smallBlindPlayer.Stack()
	// 	}
	// 	if err := i.round.bet(i.round.smallBlindPlayer.ID(), bet); err != nil {
	// 		log.Fatalf("playingSmallBlindState error: %s", err)
	// 	}
	// } else {
	// 	log.Fatal("playingSmallBlindState error, should never happen.")
	// }

	// // go to next states
	// i.round.advancePlayer()
	// i.round.smallBlindPlayer.actionRequiredThisRound = !i.round.smallBlindPlayer.allin
	// i.round.setState(i.round.roundBigBlind, true, false)
	// i.round.SetMinBetThisRound(i.round.bigBlind)

	i.table.setState(i.table.playingBigBlindState)
	return nil
}
