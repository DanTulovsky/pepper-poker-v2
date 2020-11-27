package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/dustin/go-humanize"
)

func (t *Table) bet(p *player.Player, bet int64) error {
	if bet > p.Money().Stack() {
		return fmt.Errorf("not enough money to bet $%v; have: $%v", humanize.Comma(bet), humanize.Comma(p.Money().Stack()))
	}
	if bet != p.Money().Stack() && p.Money().BetThisRound()+bet < t.minBetThisRound {
		return fmt.Errorf("player must bet minimum of %v", t.minBetThisRound)
	}
	if bet < 0 {
		return fmt.Errorf("bet cannot be < 0 (sent: %v)", bet)
	}
	if bet == 0 {
		return fmt.Errorf("cannot bet $0, call() instead")
	}

	m := p.Money()

	m.SetStack(m.Stack() - bet)
	m.SetBetThisRound(m.BetThisRound() + bet)
	p.GoAllIn(m.Stack() == 0)
	p.SetActionRequired(false)

	t.pot.Add(p.ID.String(), bet, p.AllIn())

	if p.Money().BetThisRound() > t.minBetThisRound {
		t.minBetThisRound = p.Money().BetThisRound()

		// reset any players that have put in less than this so they get to go again
		for _, p := range t.ActivePlayers() {
			if !p.AllIn() && !p.Folded() && p.Money().BetThisRound() < t.minBetThisRound {
				p.SetActionRequired(true)
			}
		}
	}

	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) call(p *player.Player) error {
	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) check(p *player.Player) error {
	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) fold(p *player.Player) error {
	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}
