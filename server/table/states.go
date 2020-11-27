package table

import (
	"fmt"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// state is the state machine for the table
type state interface {
	AddPlayer(player *player.Player) (pos int, err error)
	AvailableToJoin() bool

	Bet(p *player.Player, bet int64) error
	Check(p *player.Player) error
	Call(p *player.Player) error
	Fold(*player.Player) error

	Init() error
	Name() ppb.GameState
	Reset()
	Tick() error
	WaitingTurnPlayer() *player.Player
}

// baseState for common functions
type baseState struct {
	name  ppb.GameState
	table *Table

	l *logger.Logger
}

// newBaseState returns a new base state
func newBaseState(name ppb.GameState, table *Table) baseState {
	r := baseState{
		name:  name,
		table: table,
	}

	r.l = logger.New(name.String(), color.New(color.FgGreen))

	return r
}

// Init runs once when the stats starts
func (i *baseState) Init() error {
	return nil
}

// Name returns the name
func (i *baseState) Name() ppb.GameState {
	return i.name
}

// Resets restes for next round
func (i *baseState) Reset() {
}

// WaitingTurnPlayer returns the player whose turn it is.
func (i *baseState) WaitingTurnPlayer() *player.Player {
	p := i.table.positions[i.table.currentTurn]

	if p != nil && p.ActionRequired() {
		return p
	}

	return nil
}

// AvailableToJoin returns true if the table has empty positions
func (i *baseState) AvailableToJoin() bool {
	return false
}

func (i *baseState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("cannot add player right now")
}

// Bet processes the bet request
func (i *baseState) Bet(p *player.Player, bet int64) error {
	if i.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")
	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}
	if p.Folded() {
		return fmt.Errorf("player [%v] folded, not allowed to bet", p.Name)
	}
	if p.AllIn() {
		return fmt.Errorf("player [%v] all in, not allowed to bet", p.Name)
	}
	if bet > p.Money().Stack() {
		return fmt.Errorf("not enough money to bet $%v; have: $%v", humanize.Comma(bet), humanize.Comma(p.Money().Stack()))
	}
	if bet != p.Money().Stack() && p.Money().BetThisRound()+bet < i.table.minBetThisRound {
		return fmt.Errorf("player must bet minimum of %v", i.table.minBetThisRound)
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

	i.table.pot.Add(p.ID.String(), bet, p.AllIn())

	if p.Money().BetThisRound() > i.table.minBetThisRound {
		i.table.minBetThisRound = p.Money().BetThisRound()

		// reset any players that have put in less than this so they get to go again
		for _, p := range i.table.ActivePlayers() {
			if !p.AllIn() && !p.Folded() && p.Money().BetThisRound() < i.table.minBetThisRound {
				p.SetActionRequired(true)
			}
		}
	}

	p.SetActionRequired(false)
	p.CurrentTurn++

	return i.table.bet(p, bet)
}

// Check process the check request
func (i *baseState) Check(p *player.Player) error {
	if i.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")
	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}
	return i.table.check(p)
}

// Call process the call request
func (i *baseState) Call(p *player.Player) error {
	if i.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")
	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}
	return i.table.call(p)
}

// Fold processes the fold request
func (i *baseState) Fold(p *player.Player) error {
	if i.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")
	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}
	return i.table.fold(p)
}
