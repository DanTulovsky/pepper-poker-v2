package manager

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/proto"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/DanTulovsky/pepper-poker-v2/server/server"
	"github.com/DanTulovsky/pepper-poker-v2/server/table"
	"github.com/fatih/color"
)

var (
	numTables = 1
)

// Manager manages incoming user requests and sends them to each table
type Manager struct {
	l                  *logger.Logger
	fromGrpcServerChan chan actions.PlayerAction
	// TODO: Rename string to "playerID" type

	tables  map[id.TableID]*table.Table
	players map[id.PlayerID]*player.Player
}

// New returns a new manager
func New() *Manager {
	// channel for servers to send data to the manager
	fromServerChan := make(chan actions.PlayerAction)

	return &Manager{
		l:                  logger.New("manager", color.New(color.FgRed)),
		fromGrpcServerChan: fromServerChan,
		tables:             make(map[id.TableID]*table.Table),
		players:            make(map[id.PlayerID]*player.Player),
	}
}

// Run is the main function that starts running the entire system
func (m *Manager) Run(ctx context.Context) error {
	m.startServers(ctx, m.fromGrpcServerChan)
	m.createTables()
	m.startTables()

	m.l.Info("Starting manager loop...")
	for {
		if err := m.tick(); err != nil {
			return err
		}
	}
}

func (m *Manager) startTables() {
	for _, t := range m.tables {
		m.l.Infof("Starting table [%v]", t.Name)
		go t.Run()
	}
}

func (m *Manager) createTables() {
	m.l.Infof("Creating %v tables...", numTables)
	for i := 0; i < numTables; i++ {
		t := m.createTable()
		m.tables[t.ID] = t
	}
}

func (m *Manager) createTable() *table.Table {
	ta := make(chan actions.TableActionRequest)
	tar := make(chan actions.TableActionResult)
	return table.New(ta, tar)
}

// startServers start grpc and http servers
func (m *Manager) startServers(ctx context.Context, serverChan chan actions.PlayerAction) {
	m.l.Info("Starting gRPC and HTTP server...")

	go func() {
		if err := server.Run(ctx, serverChan); err != nil {
			m.l.Fatal(err)
		}
	}()
}

// tick is one ass through the manager
func (m *Manager) tick() error {
	m.l.Debug("Tick()")

	m.processPlayerRequests()

	time.Sleep(time.Millisecond * 500)

	return nil
}

func (m *Manager) tickTables() error {
	for _, t := range m.tables {
		// TODO: This stops the entire manager due to one broken table
		if err := t.Tick(); err != nil {
			return err
		}
	}
	return nil
}

// processPlayerRequests processes requests sent from the player via the grpc server
func (m *Manager) processPlayerRequests() {
	select {
	case in := <-m.fromGrpcServerChan:
		playerName := in.Data.PlayerName
		// playerID := in.Data.PlayerID
		playerAction := in.Data.PlayerAction

		m.l.Infof("[%v] Received request from player: %#v", playerName, playerAction.String())

		switch in.Data.PlayerAction {
		case proto.PlayerAction_PlayerActionNone:
			m.l.Infof("[%v] player sent empty action...", playerName)

		case proto.PlayerAction_PlayerActionRegister:
			if err := m.addPlayer(in); err != nil {
				m.l.Error(err)
				break
			}
		case proto.PlayerAction_PlayerActionJoinTable:
			var pos int
			var err error
			var tableID id.TableID
			if tableID, pos, err = m.joinTable(in); err != nil {
				m.l.Error(err)
				break
			}
			m.l.Infof("[%v] joined table [%v] at position [%v]", playerName, tableID, pos)
		default:
			// m.l.Infof("[%v] Doing action: %v", playerName, playerAction.String())
		}

	default:
	}
}

func (m *Manager) joinTable(in actions.PlayerAction) (tableID id.TableID, pos int, err error) {
	var t *table.Table

	// find available table
	// TODO: Handle joining a specific table (in.TableID)
	t, err = m.firstAvailableTable()
	if err != nil {
		return
	}

	playerID := id.PlayerID(in.Data.PlayerID)
	var p *player.Player
	var ok bool
	if p, ok = m.players[playerID]; !ok {
		return "", -1, fmt.Errorf("must register first")
	}

	pos, err = t.AddPlayer(p)
	return t.ID, pos, err
}

func (m *Manager) firstAvailableTable() (*table.Table, error) {
	for _, t := range m.tables {
		if t.AvailableToJoin() {
			return t, nil
		}
	}

	return nil, fmt.Errorf("unable to find free table")
}

// addPlayer add the player to the manager instance and make them available for playing games
// TODO: In the future this can pull players from a data store with proper auth
func (m *Manager) addPlayer(in actions.PlayerAction) error {
	id := id.PlayerID(in.Data.PlayerID)
	playerName := in.Data.PlayerName

	if _, ok := m.players[id]; !ok {
		m.l.Infof("[%v] Adding player to manager (id: %v)", playerName, id)
		if in.ToManagerChan == nil {
			log.Fatalf("[%v] null manager channel for player", playerName)
		}
		p := player.New(playerName, in.ToManagerChan)
		m.players[id] = p
		return nil
	}

	return fmt.Errorf("[%v] player with id %v already registered", playerName, id)
}
