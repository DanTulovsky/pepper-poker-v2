package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/dustin/go-humanize"
)

var (
	playerActions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pepperpoker_player_actions_total",
		Help: "The total number of player actions",
	}, []string{"username", "action"}) // TODO: This is only ok for very few players
)

// bet bets, 'a' is used to keep track of stats only
func (t *Table) bet(p *player.Player, bet int64, a Action) error {
	if bet > p.Money().Stack() {
		return fmt.Errorf("not enough money to bet $%v; have: $%v", humanize.Comma(bet), humanize.Comma(p.Money().Stack()))
	}
	if bet != p.Money().Stack() && p.Money().BetThisRound()+bet < t.minBetThisRound {
		return fmt.Errorf("player must bet minimum of %v", t.minBetThisRound)
	}
	if bet < 0 {
		return fmt.Errorf("bet cannot be < 0 (sent: %v)", bet)
	}

	m := p.Money()

	m.SetStack(m.Stack() - bet)
	m.SetBetThisRound(m.BetThisRound() + bet)
	p.GoAllIn(m.Stack() == 0)
	p.SetActionRequired(false)

	if bet != 0 {
		t.pot.Add(p.ID, bet, p.AllIn())

		if p.Money().BetThisRound() > t.minBetThisRound {
			t.minBetThisRound = p.Money().BetThisRound()

			// reset any players that have put in less than this so they get to go again
			for _, p := range t.ActivePlayers() {
				if !p.AllIn() && !p.Folded() && p.Money().BetThisRound() < t.minBetThisRound {
					p.SetActionRequired(true)
				}
			}
		}
	}

	// Success
	if a == ActionBet {
		playerActions.WithLabelValues(p.Username, "bet").Inc()
	}
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) call(p *player.Player) error {
	bet := t.minBetThisRound - p.Money().BetThisRound()
	if bet == 0 {
		return fmt.Errorf("no bet is needed to call, should check instead")
	}

	playerActions.WithLabelValues(p.Username, "call").Inc()
	return t.bet(p, bet, ActionCall)
}

func (t *Table) buyin(p *player.Player) error {
	if p.Money().Bank() < t.buyinAmount {
		return fmt.Errorf("table buyin is [$%v], player has: $%v", humanize.Comma(t.buyinAmount), humanize.Comma(p.Money().Stack()))
	}

	stack := p.Money().Stack() + t.buyinAmount
	bank := p.Money().Bank() - t.buyinAmount
	p.Money().SetStack(stack)
	p.Money().SetBank(bank)

	playerActions.WithLabelValues(p.Username, "buyin").Inc()
	return nil
}

func (t *Table) allin(p *player.Player) error {
	playerActions.WithLabelValues(p.Username, "allin").Inc()
	return t.bet(p, p.Money().Stack(), ActionAllIn)
}

func (t *Table) check(p *player.Player) error {
	playerActions.WithLabelValues(p.Username, "check").Inc()
	return t.bet(p, 0, ActionCheck)
}

func (t *Table) fold(p *player.Player) error {
	p.Fold()
	playerActions.WithLabelValues(p.Username, "fold").Inc()

	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}
