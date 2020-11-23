package manager

import (
	"context"
	"fmt"
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

	m.l.Info("Starting manager loop...")
	for {
		if err := m.tick(); err != nil {
			return err
		}
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
	m.l.Info("Tick()")

	m.processPlayerRequests()
	// m.processPlayerResponses()
	m.tickTables()

	time.Sleep(time.Second)

	return nil
}

func (m *Manager) tickTables() {
	for _, t := range m.tables {
		t.Tick()
	}
}

// // processPlayerResponses process responses sent to each player's channel (by the table)
// func (m *Manager) processPlayerResponses() {

// 	for _, p := range m.players {
// 	OUTER:
// 		for {
// 			select {
// 			case in := <-p.CommChannel:

// 			default:
// 				break OUTER
// 			}

// 		}
// 	}

// }

// processPlayerRequests processes requests sent from the player via the grpc server
func (m *Manager) processPlayerRequests() {
	select {
	case in := <-m.fromGrpcServerChan:
		m.l.Infof("Received request from player: %#v", in.Data.PlayerAction.String())

		switch in.Data.PlayerAction {
		case proto.PlayerAction_PlayerActionNone:
			m.l.Info("player sent empty action...")
		case proto.PlayerAction_PlayerActionRegister:
			if err := m.addPlayer(in); err != nil {
				m.l.Error(err)
			}
		case proto.PlayerAction_PlayerActionJoinTable:
			if err := m.joinTable(in); err != nil {
				m.l.Error(err)
			}
		default:
			m.l.Infof("Doing action: %v", in.Data.PlayerAction.String())
		}

	default:
	}
}

func (m *Manager) joinTable(in actions.PlayerAction) error {
	// find available table
	t, err := m.firstAvailableTable()
	if err != nil {
		return err
	}

	id := id.PlayerID(in.Data.PlayerID)
	var p *player.Player
	var ok bool
	if p, ok = m.players[id]; !ok {
		return fmt.Errorf("must register first")
	}
	return t.AddPlayer(p)
}

func (m *Manager) firstAvailableTable() (*table.Table, error) {
	for _, t := range m.tables {
		if t.State == table.TableStateWaitingPlayers {
			return t, nil
		}
	}

	return nil, fmt.Errorf("unable to find free table")
}

// addPlayer add the player to the manager instance and make them available for playing games
// TODO: In the future this can pull players from a data store with proper auth
func (m *Manager) addPlayer(in actions.PlayerAction) error {
	id := id.PlayerID(in.Data.PlayerID)

	if _, ok := m.players[id]; !ok {
		m.l.Infof("Adding player to manager: %v (%v)", in.Data.PlayerName, in.Data.PlayerID)
		p := player.New(in.Data.PlayerName, in.ToManagerChan)
		m.players[id] = p
	}

	return nil
}
