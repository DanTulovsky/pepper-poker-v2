package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/acks"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/Pallinder/go-randomdata"
	"github.com/fatih/color"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// Table hosts a game and allows playing multiple rounds
type Table struct {
	Name string
	ID   id.TableID

	// Table listens to manager actions on this channel
	TableAction chan ActionRequest

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

	// acks are used to get clients to ack at specific points in time (e.g. game start)
	currentAckToken   *acks.Token
	defaultAckTimeout time.Duration

	// index into the positions array
	currentTurn int
	// the current button, index into positions array
	button               int
	bigBlind, smallblind int64
	minBetThisRound      int64

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
func New(tableAction chan ActionRequest) *Table {
	t := &Table{
		ID:          id.NewTableID(),
		Name:        randomdata.SillyName(),
		TableAction: tableAction,
		l:           logger.New("table", color.New(color.FgYellow)),

		maxPlayers: 7,
		minPlayers: 2,

		button:     -1,
		smallblind: 5,
		bigBlind:   10,

		defaultAckTimeout: time.Second * 10,
		playerTimeout:     time.Second * 120,
		gameEndDelay:      time.Second * 10,
		gameWaitTimeout:   time.Second * 5,
		stateAdvanceDelay: time.Second * 1,
	}

	t.positions = make([]*player.Player, t.maxPlayers)

	t.waitingPlayersState = &waitingPlayersState{
		baseState:       newBaseState(ppb.GameState_GameStateWaitingPlayers, t),
		gameWaitTimeout: t.gameWaitTimeout,
	}
	t.initializingState = &initializingState{
		baseState: newBaseState(ppb.GameState_GameStateInitializing, t),
	}
	t.readyToStartState = &readyToStartState{
		baseState:     newBaseState(ppb.GameState_GameStateReadyToStart, t),
		playerTimeout: t.playerTimeout,
	}
	t.playingSmallBlindState = &playingSmallBlindState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingSmallBlind, t),
	}
	t.playingBigBlindState = &playingBigBlindState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingBigBlind, t),
	}
	t.playingPreFlopState = &playingPreFlopState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingPreFlop, t),
	}
	t.playingFlopState = &playingFlopState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingFlop, t),
	}
	t.playingTurnState = &playingTurnState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingTurn, t),
	}
	t.playingRiverState = &playingRiverState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingRiver, t),
	}
	t.playingDoneState = &playingDoneState{
		baseState: newBaseState(ppb.GameState_GameStatePlayingDone, t),
	}
	t.finishedState = &finishedState{
		baseState:    newBaseState(ppb.GameState_GameStateFinished, t),
		gameEndDelay: t.gameEndDelay,
	}

	t.State = t.waitingPlayersState

	return t
}

// Run runs the table
func (t *Table) Run() error {
	t.l.Infof("Table [%v] starting run loop...", t.Name)
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

	t.processManagerActions()

	if err := t.State.Tick(); err != nil {
		return err
	}

	t.sendUpdateToPlayers()
	return nil
}

// setAckToken sets an ack for the table
func (t *Table) setAckToken(tok *acks.Token) {
	t.currentAckToken = tok

}

// clearAckToken clears the ack
func (t *Table) clearAckToken() {
	t.currentAckToken = nil

}

// processManagerActions checks the channel from the manager for any player actions
func (t *Table) processManagerActions() error {
	select {
	case in := <-t.TableAction:
		t.l.Debugf("Received table action: %v", in.Action)
		t.processManagerAction(in)
	default:
	}

	return nil
}

func (t *Table) processManagerAction(in ActionRequest) {
	var res ActionResult

	switch in.Action {
	case ActionAddPlayer:
		pos, err := t.addPlayer(in.Player)
		switch err {
		case nil:
			res = NewTableActionResult(nil, ActionAddPlayerResult{
				Position: pos,
			})
		default:
			res = NewTableActionResult(err, nil)
		}

	case ActionRegisterPlayerCC:
		cc := in.Opts.(chan actions.GameData)
		err := t.registerPlayerCC(in.Player, cc)
		res = NewTableActionResult(err, nil)

	case ActionInfo:
		i := t.info()
		res = NewTableActionResult(nil, i)

	case ActionAckToken:
		token := in.Opts.(string)
		err := t.ackToken(in.Player, token)
		res = NewTableActionResult(err, nil)

	case ActionCheck:
		t.State.Check(in.Player)

	}
	// send reply back to manager
	in.resultChan <- res
}

func (t *Table) ackToken(p *player.Player, token string) error {

	if t.currentAckToken.String() != token {
		return fmt.Errorf("current token is [%v], sent token is [%v]", t.currentAckToken, token)
	}

	return t.currentAckToken.Ack(p)
}

func (t *Table) registerPlayerCC(p *player.Player, cc chan actions.GameData) error {

	if cc == nil {
		return fmt.Errorf("nil comm channel for player")
	}
	p.CommChannel = cc

	return nil
}

// info returns table info
func (t *Table) info() ActionInfoResult {
	return ActionInfoResult{
		AvailableToJoin: t.AvailableToJoin(),
		Name:            t.Name,
		MaxPlayers:      t.maxPlayers,
		MinPlayers:      t.minPlayers,
	}

}

// infoproto returns t.info() in a proto to send to the client
func (t *Table) infoproto() *ppb.GameInfo {
	i := t.info()
	gi := &ppb.GameInfo{
		TableName: i.Name,
		TableID:   t.ID.String(),

		GameState: t.State.Name(),

		MaxPlayers: int64(i.MaxPlayers),
		MinPlayers: int64(i.MinPlayers),
	}

	if t.currentAckToken != nil {
		gi.AckToken = t.currentAckToken.String()
	}

	return gi
}

// infoProto returns the ppb.GameData proto filled in from t.info()
func (t *Table) gameDataProto(p *player.Player) *ppb.GameData {

	// p is the player the info is being sent to
	current := t.State.WaitingTurnPlayer()

	d := &ppb.GameData{
		Info:     t.infoproto(),
		PlayerID: p.ID.String(),
	}

	if current != nil {
		d.WaitTurnID = current.ID.String()
	}

	return d
}

// advancePlayer advances t.currentPlayer to the next player
func (t *Table) advancePlayer() {
	next := t.playerAfter(t.currentTurn)
	from := t.positions[t.currentTurn].Name
	to := t.positions[next].Name

	t.l.Infof("Advancing player: %v -> %v", from, to)
	t.currentTurn = next
}

// AvailableToJoin returns true if the table has empty positions
func (t *Table) AvailableToJoin() bool {
	return t.State.AvailableToJoin()
}

// playerAfter returns the index of the first non-empty chair after index.
func (t *Table) playerAfter(index int) int {
	for i := 0; i < t.maxPlayers; i++ {
		index = (index + 1) % t.maxPlayers
		if t.positions[index] != nil {
			break
		}
	}
	return index
}

// canAdvanceState returns true if the state can advance
func (t *Table) canAdvanceState() bool {
	for _, p := range t.ActivePlayers() {
		if p.ActionRequired() {
			return false
		}
	}
	return true
}

// setState sets the state of the table
func (t *Table) setState(s state) {
	var from, to string = "nil", "nil"
	from = t.State.Name().String()
	to = s.Name().String()

	t.l.Infof(color.GreenString("Changing State (%v): %v -> %v"), t.stateAdvanceDelay, from, to)
	// TODO: This blocks all processing and is really not needed
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

// addPlayer adds a player to the table
func (t *Table) addPlayer(p *player.Player) (int, error) {
	return t.State.AddPlayer(p)
}

func (t *Table) nextAvailablePosition() int {
	for i, p := range t.positions {
		if p == nil {
			return i
		}
	}
	return -1
}

// SetPlayersActionRequired resets the actionRequired attribute on players before each state
func (t *Table) SetPlayersActionRequired() {
	for _, p := range t.ActivePlayers() {
		if !p.AllIn() && !p.Folded() {
			p.SetActionRequired(true)
		}
	}
}

// sendUpdateToPlayers sends updates to players as needed
func (t *Table) sendUpdateToPlayers() {
	for _, p := range t.ActivePlayers() {
		in := t.gameDataProto(p)
		action := actions.NewGameData(in)
		if p.CommChannel == nil {
			t.l.Debugf("player [%v] has nil comm channel", p.Name)
			continue
		}
		select {
		case p.CommChannel <- action:
		default:
		}
	}
}

// playersReady returns true if all players are ready
func (t *Table) playersReady() bool {
	for _, p := range t.ActivePlayers() {
		if p.CommChannel == nil {
			return false
		}
	}
	return true
}
