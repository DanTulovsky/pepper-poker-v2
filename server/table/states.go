package table

import (
	"fmt"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/fatih/color"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	statesEntered = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pepperpoker_states_entered_total",
		Help: "The total number of states entered",
	}, []string{"state"})
)

// state is the state machine for the table
type state interface {
	AddPlayer(player *player.Player) (pos int, err error)
	AvailableToJoin() bool

	Bet(p *player.Player, bet int64) error
	Check(p *player.Player) error
	Call(p *player.Player) error
	Fold(*player.Player) error
	AllIn(*player.Player) error
	BuyIn(*player.Player) error

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
	statesEntered.WithLabelValues(i.Name().String()).Inc()

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

// AddPlayer adds the player to the table
// In all states but the first, this puts the player in a list of pending players
func (i *baseState) AddPlayer(p *player.Player) (pos int, err error) {
	if p != nil {
		i.table.pendingPlayers = append(i.table.pendingPlayers, p)
		return -1, nil
	}

	return -1, fmt.Errorf("nil player passed to AddPlayer")
}

// AllIn process the allin request
func (i *baseState) AllIn(p *player.Player) error {
	if i.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")
	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}
	return i.table.allin(p)
}

// BuyIn process the buyin request. The player must already be sitting (have position) at the table
func (i *baseState) BuyIn(p *player.Player) error {

	if p.TablePosition < 0 {
		return fmt.Errorf("must JoinTable before tyring to buyin")
	}

	return i.table.buyin(p)
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

	return i.table.bet(p, bet, actions.ActionBet)
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
