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

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
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

func (m *Manager) createTables() {
	m.l.Infof("Creating %v tables...", numTables)
	for i := 0; i < numTables; i++ {
		t := m.createTable()
		m.tables[t.ID] = t
	}
}

func (m *Manager) createTable() *table.Table {
	ta := make(chan table.ActionRequest)
	return table.New(ta)
}

func (m *Manager) startTables() {
	for _, t := range m.tables {
		m.l.Infof("Starting table [%v]", t.Name)
		go t.Run()
	}
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

// processPlayerRequests processes requests sent from the player via the grpc server
func (m *Manager) processPlayerRequests() {

	var p *player.Player
	var err error

	select {
	case in := <-m.fromGrpcServerChan:
		playerName := in.ClientInfo.PlayerName
		playerID := id.PlayerID(in.ClientInfo.PlayerID)
		tableID := id.TableID(in.ClientInfo.TableID)
		playerAction := in.Action

		m.l.Infof("[%v] Received request from player: %#v", playerName, playerAction.String())

		switch playerAction {
		case proto.PlayerAction_PlayerActionNone:
			m.l.Infof("[%v] player sent empty action...", playerName)
			break

		case proto.PlayerAction_PlayerActionRegister:
			if p, err = m.addPlayer(in); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.RegisterResponse{
				PlayerID: p.ID.String(),
			})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionJoinTable:
			var pos int
			var tableID id.TableID
			if tableID, pos, err = m.joinTable(in); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			if p, err = m.playerByID(playerID); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.JoinTableResponse{
				TableID:  tableID.String(),
				Position: int64(pos),
			})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionCheck:
			if err := m.playerCheck(playerID, tableID); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.TakeTurnResponse{})
			in.ResultC <- result

		default:
			// m.l.Infof("[%v] Doing action: %v", playerName, playerAction.String())
		}

	default:
	}
}

// playerByID returns the player by ID
func (m *Manager) playerByID(playerID id.PlayerID) (*player.Player, error) {
	if p, ok := m.players[playerID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("player with id [%v] not found", playerID)
}

// playerCheck sends the Check action to the table
func (m *Manager) playerCheck(playerID id.PlayerID, tableID id.TableID) error {
	t := m.tables[tableID]
	p := m.players[playerID]

	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionCheck, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// jointable attempts to join a table
func (m *Manager) joinTable(in actions.PlayerAction) (tableID id.TableID, pos int, err error) {
	var t *table.Table
	pos = -1

	// find available table
	// TODO: Handle joining a specific table (in.TableID)
	t, err = m.firstAvailableTable()
	if err != nil {
		return
	}

	playerID := id.PlayerID(in.ClientInfo.PlayerID)
	var p *player.Player
	var ok bool
	if p, ok = m.players[playerID]; !ok {
		err = fmt.Errorf("must register first")
		return
	}

	// Table response comes back over this channel
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionAddPlayer, result, p, nil)
	t.TableAction <- req

	// block(!?) until table responds
	res := <-result
	err = res.Err
	if err != nil {
		return
	}

	r := res.Result.(table.ActionAddPlayerResult)
	return t.ID, r.Position, err
}

// firstAvailableTable returns the first table with an empty spot for the player
func (m *Manager) firstAvailableTable() (*table.Table, error) {
	for _, t := range m.tables {

		result := make(chan table.ActionResult)
		req := table.NewTableAction(table.ActionInfo, result, nil, nil)
		t.TableAction <- req

		res := <-result
		err := res.Err
		if err != nil {
			return nil, err
		}

		r := res.Result.(table.ActionInfoResult)
		if r.AvailableToJoin {
			return t, nil
		}
	}

	return nil, fmt.Errorf("unable to find free table")
}

// addPlayer add the player to the manager instance and make them available for playing games
func (m *Manager) addPlayer(in actions.PlayerAction) (*player.Player, error) {
	id := id.NewPlayerID()
	playerName := in.ClientInfo.PlayerName

	if _, ok := m.players[id]; !ok {
		m.l.Infof("[%v] Adding player to manager (id: %v)", playerName, id)
		if in.ToManagerChan == nil {
			log.Fatalf("[%v] null manager channel for player", playerName)
		}
		p := player.New(playerName, in.ToManagerChan)
		m.players[id] = p
		return p, nil
	}

	return nil, fmt.Errorf("[%v] player with id %v already registered", playerName, id)
}
