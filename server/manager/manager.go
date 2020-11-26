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

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	numTables = 1
)

// Manager manages incoming user requests and sends them to each table
type Manager struct {
	l                  *logger.Logger
	fromGrpcServerChan chan actions.PlayerAction

	tables map[id.TableID]*table.Table
	// Todo: Consider either adding locks on *Player or just uding IDs here
	// The Table accesses and calls methods on Player
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
	for i, t := range m.tables {
		m.l.Infof("Starting table [%v]", t.Name)
		go func(t *table.Table, i id.TableID) {
			if err := t.Run(); err != nil {
				m.l.Errorf("Table [%v] returned error: %v", i, err)
			}
		}(t, i)
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
	var t *table.Table
	var err error

	select {
	case in := <-m.fromGrpcServerChan:
		playerName := in.ClientInfo.PlayerName
		playerID := id.PlayerID(in.ClientInfo.PlayerID)
		tableID := id.TableID(in.ClientInfo.TableID)
		playerAction := in.Action

		if playerID != "" {
			if p, err = m.playerByID(playerID); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				return
			}
		}

		if tableID != "" {
			if t, err = m.tableByID(tableID); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				return

			}
		}

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
			if tableID, pos, err = m.joinTable(p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.JoinTableResponse{
				TableID:  tableID.String(),
				Position: int64(pos),
			})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionPlay:
			if err := m.registerPlayerCC(p, t, in.ToClientChan); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.JoinTableResponse{})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionAckToken:
			if err := m.playerAckToken(p, t, in.Opts.AckToken); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.AckTokenResponse{})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionCheck:
			if err := m.playerCheck(p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.TakeTurnResponse{})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionFold:
			if err := m.playerFold(p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.TakeTurnResponse{})
			m.l.Info("Sending reply back to player")
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionCall:
			if err := m.playerCall(p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.TakeTurnResponse{})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionAllIn:
			if err := m.playerAllIn(p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.TakeTurnResponse{})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionBet:
			amount := in.Opts.GetBetAmount()
			if err := m.playerBet(p, t, amount); err != nil {
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

// playerAckToken acks the token for the player at the table
func (m *Manager) playerAckToken(p *player.Player, t *table.Table, token string) error {
	// Table response comes back over this channel
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionAckToken, result, p, token)

	t.TableAction <- req

	// block(!?) until table responds
	res := <-result
	return res.Err
}

// registerPlayerCC registers the player channel and starts streaming game data to it
func (m *Manager) registerPlayerCC(p *player.Player, t *table.Table, cc chan actions.GameData) error {
	// Table response comes back over this channel
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionRegisterPlayerCC, result, p, cc)

	t.TableAction <- req

	// block(!?) until table responds
	res := <-result
	return res.Err
}

// tableByID returns the table with the given id
func (m *Manager) tableByID(tableID id.TableID) (*table.Table, error) {
	if t, ok := m.tables[tableID]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("cannot find table with id [%v]", tableID)
}

// playerByID returns the player by ID
func (m *Manager) playerByID(playerID id.PlayerID) (*player.Player, error) {
	if p, ok := m.players[playerID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("player with id [%v] not found", playerID)
}

// playerCheck sends the Check action to the table
func (m *Manager) playerCheck(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionCheck, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerFold sends the Fold action to the table
func (m *Manager) playerFold(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionFold, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerCall sends the Call action to the table
func (m *Manager) playerCall(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionCall, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerAllIn sends the AllIn action to the table
func (m *Manager) playerAllIn(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionAllIn, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerBet sends the Bet action to the table
func (m *Manager) playerBet(p *player.Player, t *table.Table, amount int64) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(table.ActionBet, result, p, amount)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// jointable attempts to join a table
func (m *Manager) joinTable(p *player.Player, t *table.Table) (tableID id.TableID, pos int, err error) {
	pos = -1

	// find available table
	if t == nil {
		t, err = m.firstAvailableTable()
		if err != nil {
			return
		}
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
	playerName := in.ClientInfo.PlayerName
	if m.havePlayerName(playerName) {
		return nil, fmt.Errorf("already have player with name [%v]", playerName)
	}

	m.l.Infof("[%v] Adding player to manager...", playerName)
	p := player.New(playerName)
	m.players[p.ID] = p
	return p, nil
}

// havePlayerName returns true if there is a player with the given name
func (m *Manager) havePlayerName(name string) bool {
	for _, p := range m.players {
		if p.Name == name {
			return true
		}
	}
	return false
}
