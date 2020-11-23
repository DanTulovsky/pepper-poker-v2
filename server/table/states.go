package table

import (
	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/fatih/color"
)

// state is the state machine for the table
type state interface {
	AddPlayer(player *player.Player) (pos int, err error)
	AvailableToJoin() bool
	Init()
	Name() string
	Reset()
	StartGame() error
	Tick() error
	WhoseTurn() *player.Player
}

// baseState for common functions
type baseState struct {
	name  string
	table *Table

	l *logger.Logger
}

// newBaseState returns a new base state
func newBaseState(name string, table *Table) baseState {
	r := baseState{
		name:  name,
		table: table,
	}

	r.l = logger.New(name, color.New(color.FgGreen))

	return r
}

// Init runs once when the stats starts
func (i *baseState) Init() {

}

// Name returns the name
func (i *baseState) Name() string {
	return i.name
}

// Resets restes for next round
func (i *baseState) Reset() {
}

// WhoseTurn returns the player whose turn it is.
func (i *baseState) WhoseTurn() *player.Player {
	return nil
}

// AvailableToJoin returns true if the table has empty positions
func (i *baseState) AvailableToJoin() bool {
	return false
}
