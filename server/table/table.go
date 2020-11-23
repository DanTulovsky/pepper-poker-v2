package table

import (
	"fmt"
	"math/rand"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/fatih/color"
)

// State is the current state of the table
type State int

const (
	// TableStateNotReady ...
	TableStateNotReady State = iota
	// TableStateWaitingPlayers ...
	TableStateWaitingPlayers
	// TableStateReady ..
	TableStateReady
	// TableStatePlaying ...
	TableStatePlaying
	// TableStateDone ...
	TableStateDone
)

// Table hosts a game and allows playing multiple rounds
type Table struct {
	Name string
	ID   id.TableID

	tableAction       chan actions.TableActionRequest
	tableActionResult chan actions.TableActionResult

	players    map[id.PlayerID]*player.Player
	maxPlayers int
	minPlayers int

	State State

	l *logger.Logger
}

// New creates a new table
func New(tableAction chan actions.TableActionRequest, tableActionResult chan actions.TableActionResult) *Table {
	return &Table{
		ID:                id.NewTableID(),
		tableAction:       tableAction,
		tableActionResult: tableActionResult,
		l:                 logger.New("table", color.New(color.FgYellow)),
		players:           make(map[id.PlayerID]*player.Player),
		maxPlayers:        7,
		minPlayers:        1,
		State:             TableStateWaitingPlayers,
	}
}

// Tick ticks the table
func (t *Table) Tick() {
	t.l.Info("Tick()")

	if len(t.players) < t.minPlayers {
		return
	}

	t.sendUpdateToPlayers()

}

// AddPlayer adds a player to the table
func (t *Table) AddPlayer(p *player.Player) error {

	if t.State != TableStateWaitingPlayers {
		return fmt.Errorf("table state not ok for adding players: %v", t.State)
	}

	if _, ok := t.players[p.ID]; !ok {
		t.players[p.ID] = p
		return nil
	}

	return fmt.Errorf("player already at the table: %v (%v)", p.Name, p.ID)
}

// sendUpdateToPlayers sends updates to players as needed
func (t *Table) sendUpdateToPlayers() {
	// TODO: read from a channel that has updates

	t.l.Info("Sending updates to players...")

	for _, p := range t.players {
		action := actions.NewManagerAction(rand.Int63n(200))
		// TODO: This should not block for when clients drop
		p.CommChannel <- action
	}

}
