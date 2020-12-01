package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingSmallBlindState struct {
	baseState
}

func (i *playingSmallBlindState) Init() error {
	i.baseState.Init()

	i.l.Infof("[%v] putting in small blind...", i.table.smallBlindPlayer.Name)

	bet := i.table.smallBlind
	if i.table.smallBlindPlayer.Money().Stack() < bet {
		bet = i.table.smallBlindPlayer.Money().Stack()
	}
	if err := i.table.bet(i.table.smallBlindPlayer, bet, ActionBet); err != nil {
		i.l.Fatalf("playingSmallBlindState error: %s", err)
	}

	i.table.advancePlayer()
	return nil
}

func (i *playingSmallBlindState) Bet(p *player.Player, bet int64) error {
	return fmt.Errorf("only small blind bets")
}

func (i *playingSmallBlindState) Call(p *player.Player) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingSmallBlindState) Check(p *player.Player) error {
	return fmt.Errorf("cannot call during this round")
}

func (i *playingSmallBlindState) Fold(p *player.Player) error {
	return fmt.Errorf("cannot fold during this round")
}

func (i *playingSmallBlindState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	i.table.setState(i.table.playingBigBlindState)
	return nil
}

func (i *playingSmallBlindState) WaitingTurnPlayer() *player.Player {
	return nil
}
