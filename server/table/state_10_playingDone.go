package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type playingDoneState struct {
	baseState
}

func (i *playingDoneState) bet(p *player.Player, bet int64) error {
	return fmt.Errorf("round is done")
}

func (i *playingDoneState) call(p *player.Player) error {
	return fmt.Errorf("round is done")
}

func (i *playingDoneState) check(p *player.Player) error {
	return fmt.Errorf("round is done")
}

func (i *playingDoneState) fold(p *player.Player) error {
	return fmt.Errorf("round is done")
}

func (i *playingDoneState) Init() {
}

func (i *playingDoneState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	// already finishing, just waiting for client acks
	// if i.finishing {
	// 	i.logger.Debug("Waiting for acks...")
	// 	return nil
	// }

	// i.logger.Info("Tick()")

	// i.logger.Info("Have winner... ")

	// // Collect all the player hands.
	// var hands []*PlayerHand
	// for _, p := range i.round.players {
	// 	i.logger.Infof("[%v] => bet this hand: %v; => folded? %t", p.Name(), humanize.Comma(i.round.Pot().GetBet(p.ID())), p.folded)

	// 	if !p.folded {
	// 		cards := append(p.Hole(), i.round.board.Cards()...)
	// 		i.logger.Infof("Adding [%v] to hands to check: %v", p.Name(), cards)
	// 		ph := NewPlayerHand(p.ID(), cards)
	// 		hands = append(hands, ph)
	// 	}
	// }

	// var levels []Winners
	// switch {
	// case len(hands) == 1:
	// 	levels = []Winners{{hands[0].ID}}
	// case len(hands) > 1:
	// 	// Calculate best hands
	// 	i.logger.Info("Calculating best hands...")
	// 	levels = BestHand(hands)
	// default:
	// 	// should never happen, everyone can't fold
	// 	log.Fatal("Somehow all players managed to fold, how can that be?")
	// }
	// i.round.Pot().Finalize(levels)

	// // Set winners
	// for _, p := range i.round.players {
	// 	if !p.folded && len(hands) > 1 {
	// 		// set the player's best hand
	// 		cards := append(p.Hole(), i.round.board.Cards()...)
	// 		hand := BestCombo(cards...)
	// 		p.hand = &PlayerHand{
	// 			Hand: hand,
	// 		}
	// 	}

	// 	winnings, _ := i.round.Pot().GetWinnings(p.ID())
	// 	if winnings > 0 {
	// 		i.logger.Infof("[%v] is a Winner!", p.Name())
	// 		p.SetWinner(winnings)
	// 	}
	// }

	// i.round.LogSystemTurn(i.round.tableID, i.round.id, ppb.Action_ActionFinishRound, nil)

	// i.finishing = true
	// i.round.SetToken(uuid.New().String())
	// i.round.Acks[i.round.Token()] = []string{}
	// go i.finishAfterAck(i.round.Token())
	i.table.setState(i.table.finishedState)
	return nil
}
