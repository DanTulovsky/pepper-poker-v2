package table

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/Pallinder/go-randomdata"
	"github.com/fatih/color"
)

// Table hosts a game and allows playing multiple rounds
type Table struct {
	Name string
	ID   id.TableID

	tableAction       chan actions.TableActionRequest
	tableActionResult chan actions.TableActionResult

	// table positions
	positions []*player.Player

	maxPlayers int
	minPlayers int

	waitingPlayersState state
	initializingState   state
	readyToStartState   state

	playingSmallBlindState state
	playingBigBlindState   state
	playingPreFlopState    state
	playingFlopState       state
	playingTurnState       state
	playingRiverState      state
	playingDoneState       state
	finishedState          state

	State state

	// how long to wait for player to make a move
	playerTimeout time.Duration
	// how long to wait after game ends before starting a new one
	gameEndDelay time.Duration
	// how long to wait between state transitions
	stateAdvanceDelay time.Duration
	// how long to wait after the last payer is added before starting the game
	gameWaitTimeout time.Duration

	gameStartsInTime time.Duration

	l *logger.Logger
}

// New creates a new table
func New(tableAction chan actions.TableActionRequest, tableActionResult chan actions.TableActionResult) *Table {
	t := &Table{
		ID:                id.NewTableID(),
		Name:              randomdata.SillyName(),
		tableAction:       tableAction,
		tableActionResult: tableActionResult,
		l:                 logger.New("table", color.New(color.FgYellow)),

		maxPlayers: 7,
		minPlayers: 1,

		playerTimeout:     time.Second * 120,
		gameEndDelay:      time.Second * 10,
		gameWaitTimeout:   time.Second * 5,
		stateAdvanceDelay: time.Second * 1,
	}

	t.positions = make([]*player.Player, t.maxPlayers)

	t.waitingPlayersState = &waitingPlayersState{
		baseState:       newBaseState("waitingPlayers", t),
		gameWaitTimeout: t.gameWaitTimeout,
	}
	t.initializingState = &initializingState{
		baseState: newBaseState("initializing", t),
	}
	t.readyToStartState = &readyToStartState{
		baseState:     newBaseState("readyToStart", t),
		playerTimeout: t.playerTimeout,
	}
	t.playingSmallBlindState = &playingSmallBlindState{
		baseState: newBaseState("playingSmallBlindState", t),
	}
	t.playingBigBlindState = &playingBigBlindState{
		baseState: newBaseState("playingBigBlindState", t),
	}
	t.playingPreFlopState = &playingPreFlopState{
		baseState: newBaseState("playingPreFlopState", t),
	}
	t.playingFlopState = &playingFlopState{
		baseState: newBaseState("playingFlopState", t),
	}
	t.playingTurnState = &playingTurnState{
		baseState: newBaseState("playingTurnState", t),
	}
	t.playingRiverState = &playingRiverState{
		baseState: newBaseState("playingRiverState", t),
	}
	t.playingDoneState = &playingDoneState{
		baseState: newBaseState("playingDoneState", t),
	}
	t.finishedState = &finishedState{
		baseState:    newBaseState("gameFinished", t),
		gameEndDelay: t.gameEndDelay,
	}

	t.State = t.waitingPlayersState

	return t
}

// Run runs the table
func (t *Table) Run() error {
	for {
		if err := t.Tick(); err != nil {
			return err
		}
		time.Sleep(time.Second)
	}

}

// Tick ticks the table
func (t *Table) Tick() error {
	t.l.Debug("Tick()")

	if err := t.State.Tick(); err != nil {
		return err
	}

	t.sendUpdateToPlayers()
	return nil
}

// AvailableToJoin returns true if the table has empty positions
func (t *Table) AvailableToJoin() bool {
	return t.State.AvailableToJoin()
}

// setState sets the state of the table
func (t *Table) setState(s state) {
	var from, to string = "nil", "nil"
	from = t.State.Name()
	to = s.Name()

	t.l.Infof(color.GreenString("Changing State (%v): %v -> %v"), t.stateAdvanceDelay, from, to)
	// TODO: This blocks all processing
	time.Sleep(t.stateAdvanceDelay)
	t.State = s

	s.Init()
}

// ActivePlayers returns the players at the table
func (t *Table) ActivePlayers() []*player.Player {
	players := []*player.Player{}

	for _, p := range t.positions {
		if p != nil {
			players = append(players, p)
		}
	}

	return players
}

// numActivePlayers returns the number of players at the table
func (t *Table) numActivePlayers() int {
	return len(t.ActivePlayers())
}

// PlayerPosition returns the position of the player
func (t *Table) PlayerPosition(p *player.Player) (int, error) {

	for i, pos := range t.positions {
		if p == pos {
			return i, nil
		}
	}

	return -1, fmt.Errorf("no such player at this table: %v", p.ID)
}

// playerAtTable returns true if the player is at this table
func (t *Table) playerAtTable(p *player.Player) bool {

	for _, pos := range t.positions {
		if p == pos {
			return true
		}
	}
	return false
}

// AddPlayer tries to add a player to the table and returns the position if successfull
func (t *Table) AddPlayer(p *player.Player) (int, error) {
	return t.State.AddPlayer(p)
}

// AddPlayer adds a player to the table
func (t *Table) addPlayer(p *player.Player) (int, error) {

	if t.State != t.waitingPlayersState {
		return -1, fmt.Errorf("table state not ok for adding players: %v", t.State)
	}

	if t.numActivePlayers() == t.maxPlayers {
		return -1, fmt.Errorf("no available positions at table")
	}

	if !t.playerAtTable(p) {
		t.positions[t.nextAvailablePosition()] = p
		return t.PlayerPosition(p)
	}

	return -1, fmt.Errorf("player already at the table: %v (%v)", p.Name, p.ID)
}

func (t *Table) nextAvailablePosition() int {
	for i, p := range t.positions {
		if p == nil {
			return i
		}
	}
	return -1
}

// sendUpdateToPlayers sends updates to players as needed
func (t *Table) sendUpdateToPlayers() {
	// TODO: read from a channel that has updates

	t.l.Debug("Sending updates to players...")

	for _, p := range t.ActivePlayers() {
		action := actions.NewManagerAction(rand.Int63n(200))
		// TODO: This should not block for when clients drop
		p.CommChannel <- action
	}

}
