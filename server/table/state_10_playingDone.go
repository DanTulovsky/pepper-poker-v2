package table

import (
	"fmt"
	"log"

	"github.com/DanTulovsky/pepper-poker-v2/poker"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/dustin/go-humanize"
)

type playingDoneState struct {
	baseState
}

func (i *playingDoneState) Init() error {
	i.baseState.Init()

	i.l.Info("Have winner!")

	// Collect all the player hands.
	var hands []*poker.PlayerHand
	for _, p := range i.table.CurrentHandPlayers() {
		i.l.Infof("[%v] => bet this hand: %v; => folded? %t", p.Name, humanize.Comma(i.table.pot.GetBet(p.ID)), p.Folded())

		if !p.Folded() {
			cards := append(p.Hole(), i.table.board.Cards()...)
			i.l.Infof("Adding [%v] to hands to check: %v", p.Name, cards)
			ph := poker.NewPlayerHand(p.ID, cards)
			hands = append(hands, ph)
		}
	}

	var levels []poker.Winners

	switch {
	case len(hands) == 1:
		levels = []poker.Winners{{hands[0].ID}}
	case len(hands) > 1:
		// Calculate best hands
		i.l.Info("Calculating best hands...")
		levels = poker.BestHand(hands)
	default:
		// should never happen, everyone can't fold
		log.Fatal("Somehow all players managed to fold, how can that be?")
	}
	i.table.pot.Finalize(levels)

	// Set winners
	for _, p := range i.table.CurrentHandPlayers() {
		p.Stats.GamesPlayedInc()

		if !p.Folded() && len(hands) > 1 {
			// set the player's best hand
			cards := append(p.Hole(), i.table.board.Cards()...)
			hand := poker.BestCombo(cards...)

			p.SetPlayerHand(&poker.PlayerHand{
				Hand: hand,
			})

			p.Stats.ComboInc(hand.Combo())
		}

		winnings, _ := i.table.pot.GetWinnings(p.ID)
		if winnings > 0 {
			i.l.Infof("[%v] is a Winner ([%v] %v)", p.Name, p.PlayerHand().Hand.Combo().String(), p.PlayerHand().Hand.Cards())

			p.Stats.GamesWonInc()
			p.SetWinnerAndWinnings(winnings)
			// Return winnings to the stack
			// TODO: On disconnect, return money to the bank
			p.Money().SetStack(p.Money().Stack() + p.Money().Winnings())
		}

		// Set money sets
		p.Stats.MoneySet("bank", p.Money().Bank())
		p.Stats.MoneySet("stack", p.Money().Stack())
		p.Stats.MoneySet("winnings", p.Money().Winnings())
		p.Stats.MoneySet("bet_this_hand", i.table.pot.GetBet(p.ID))

		// records players that reached here
		for _, p := range i.table.CurrentHandActivePlayers() {
			p.Stats.StateInc("done")
		}
	}
	return nil
}

func (i *playingDoneState) Bet(p *player.Player, bet int64) error {
	return fmt.Errorf("hand is done")
}

func (i *playingDoneState) Call(p *player.Player) error {
	return fmt.Errorf("hand is done")
}

func (i *playingDoneState) Check(p *player.Player) error {
	return fmt.Errorf("hand is done")
}

func (i *playingDoneState) Fold(p *player.Player) error {
	return fmt.Errorf("hand is done")
}

func (i *playingDoneState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	i.table.setState(i.table.finishedState)
	return nil
}

func (i *playingDoneState) WaitingTurnPlayer() *player.Player {
	return nil
}
