package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingBigBlindState struct {
	baseState
}

func (i *playingBigBlindState) Init() error {
	i.l.Infof("[%v] putting in big blind...", i.table.bigBlindPlayer.Name)

	bet := i.table.smallBlind
	if i.table.bigBlindPlayer.Money().Stack() < bet {
		bet = i.table.bigBlindPlayer.Money().Stack()
	}
	if err := i.table.bet(i.table.bigBlindPlayer, bet); err != nil {
		i.l.Fatalf("playingBigBlindState error: %s", err)
	}

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

	i.table.setState(i.table.playingPreFlopState)

	return nil
}

func (i *playingBigBlindState) WaitingTurnPlayer() *player.Player {
	return nil
}
