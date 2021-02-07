package manager

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-client-go/zipkin"
	"github.com/uber/jaeger-lib/metrics"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/proto"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/DanTulovsky/pepper-poker-v2/server/server"
	"github.com/DanTulovsky/pepper-poker-v2/server/table"
	"github.com/DanTulovsky/pepper-poker-v2/server/users"
	"github.com/fatih/color"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	tickDelay = flag.Duration("manager_tick_delay", time.Millisecond*10, "delay between manager ticks")
	numTables = 1
)

const (
	jaegerSamplingServerURL = "http://otel-collector.observability:5778/sampling"
	jaegerCollectorEndpoint = "http://otel-collector.observability:14268/api/traces"
	jaegerServiceName       = "pepper-poker"
)

// Manager manages incoming user requests and sends them to each table
type Manager struct {
	l                  *logger.Logger
	fromGrpcServerChan chan actions.PlayerAction

	tables map[id.TableID]*table.Table
	// Todo: Consider either adding locks on *Player or just uding IDs here
	// The Table accesses and calls methods on Player
	players map[id.PlayerID]*player.Player

	// Map is username to external user object
	users map[string]users.User

	defaultPlayerBank int64
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
		defaultPlayerBank:  10000,
	}
}

// Run is the main function that starts running the entire system
func (m *Manager) Run(ctx context.Context) error {
	closer, err := m.enableTracer()
	if err != nil {
		return err
	}
	defer closer.Close()

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

func (m *Manager) enableTracer() (io.Closer, error) {
	m.l.Info("Enabling OpenTracing tracer...")

	zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()
	serviceName := jaegerServiceName
	jLogger := jaegerlog.NullLogger
	jMetricsFactory := metrics.NullFactory

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		return nil, err
	}

	cfg.Reporter.CollectorEndpoint = jaegerCollectorEndpoint
	// github.com/DanTulovsky/k8s-configs/configs/jaeger/operator-config.yaml has the config
	cfg.Sampler = &jaegercfg.SamplerConfig{
		Type:              jaeger.SamplerTypeRemote,
		Param:             0, // default sampling if server does not answer
		SamplingServerURL: jaegerSamplingServerURL,
	}
	cfg.RPCMetrics = true

	// Create tracer and then initialize global tracer
	closer, err := cfg.InitGlobalTracer(
		serviceName,
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
		// jaegercfg.Injector(opentracing.HTTPHeaders, zipkinPropagator),
		// upstream from ambassador is in zipkin format
		jaegercfg.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
		jaegercfg.ZipkinSharedRPCSpan(true),
		// jaegercfg.Ta
	)

	if err != nil {
		return nil, err
	}

	return closer, nil
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

// tick is one pass through the manager
func (m *Manager) tick() error {
	// m.l.Debug("Tick()")

	m.processPlayerRequests()

	time.Sleep(*tickDelay)

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
		playerUsername := in.ClientInfo.PlayerUsername
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

		m.l.Infof("[%v] Received request from player: %#v", playerUsername, playerAction.String())

		if !users.Check(playerUsername) {
			err = fmt.Errorf("invalid username or password")
			in.ResultC <- actions.NewPlayerActionError(err)
			return
		}

		switch playerAction {
		case proto.PlayerAction_PlayerActionNone:
			m.l.Infof("[%v] player sent empty action...", playerName)
			break

		case proto.PlayerAction_PlayerActionRegister:
			if p, err = m.addPlayer(in); err != nil {
				if errors.Is(err, actions.ErrUserExists) {
					m.l.Infof("[%v] already registered...", p.Name)
					// user already registered
					result := actions.NewPlayerActionResult(nil, &ppb.RegisterResponse{
						PlayerID: p.ID.String(),
					})
					in.ResultC <- result
					break
				}
				// actual error
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			// new regisration
			result := actions.NewPlayerActionResult(err, &ppb.RegisterResponse{
				PlayerID: p.ID.String(),
			})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionDisconnect:
			if err = m.disconnectPlayer(in.Ctx, p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.DisconnectResponse{})
			in.ResultC <- result

		case proto.PlayerAction_PlayerActionJoinTable:
			var pos int
			if tableID, pos, err = m.joinTable(in.Ctx, p, t); err != nil {
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

		case proto.PlayerAction_PlayerActionBuyIn:
			if err := m.playerBuyIn(p, t); err != nil {
				m.l.Error(err)
				in.ResultC <- actions.NewPlayerActionError(err)
				break
			}
			result := actions.NewPlayerActionResult(err, &ppb.TakeTurnResponse{})
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
	req := table.NewTableAction(actions.ActionAckToken, result, p, token)

	t.TableAction <- req

	// block(!?) until table responds
	res := <-result
	return res.Err
}

// registerPlayerCC registers the player channel and starts streaming game data to it
func (m *Manager) registerPlayerCC(p *player.Player, t *table.Table, cc chan actions.GameData) error {
	// Table response comes back over this channel
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionRegisterPlayerCC, result, p, cc)

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

// playerBuyIn sends the BuyIn action to the table
func (m *Manager) playerBuyIn(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionBuyIn, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerCheck sends the Check action to the table
func (m *Manager) playerCheck(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionCheck, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerFold sends the Fold action to the table
func (m *Manager) playerFold(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionFold, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerCall sends the Call action to the table
func (m *Manager) playerCall(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionCall, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerAllIn sends the AllIn action to the table
func (m *Manager) playerAllIn(p *player.Player, t *table.Table) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionAllIn, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// playerBet sends the Bet action to the table
func (m *Manager) playerBet(p *player.Player, t *table.Table, amount int64) error {
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionBet, result, p, amount)
	t.TableAction <- req

	// block until response
	res := <-result

	return res.Err
}

// jointable attempts to join a table
func (m *Manager) joinTable(ctx context.Context, p *player.Player, t *table.Table) (tableID id.TableID, pos int, err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "joinTable")
	span.SetTag("playerUsername", p.Username)
	ext.Component.Set(span, "Manager")
	defer span.Finish()

	if p == nil {
		err = fmt.Errorf("joinTable received nil player")
		span.LogFields(log.String("error", err.Error()))
		ext.Error.Set(span, true)

		return "", -1, err
	}

	pos = -1

	// find available table
	if t == nil {
		t, err = m.firstAvailableTable()
		if err != nil {
			span.LogFields(log.String("error", err.Error()))
			ext.Error.Set(span, true)
			return
		}
	}

	// Table response comes back over this channel
	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionAddPlayer, result, p, nil)
	t.TableAction <- req

	// block(!?) until table responds
	res := <-result
	err = res.Err
	if err != nil {
		span.LogFields(log.String("error", err.Error()))
		ext.Error.Set(span, true)
		return
	}

	span.SetTag("table", t.Name)
	r := res.Result.(table.ActionAddPlayerResult)
	return t.ID, r.Position, err
}

// firstAvailableTable returns the first table with an empty spot for the player
func (m *Manager) firstAvailableTable() (*table.Table, error) {
	for _, t := range m.tables {

		result := make(chan table.ActionResult)
		req := table.NewTableAction(actions.ActionInfo, result, nil, nil)
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

// disconnectPlayer handles a player that disconnected
func (m *Manager) disconnectPlayer(ctx context.Context, p *player.Player, t *table.Table) error {

	span, _ := opentracing.StartSpanFromContext(ctx, "disconnectPlayer")
	span.SetTag("playerUsername", p.Username)
	ext.Component.Set(span, "Manager")
	defer span.Finish()

	result := make(chan table.ActionResult)
	req := table.NewTableAction(actions.ActionDisconnect, result, p, nil)
	t.TableAction <- req

	// block until response
	res := <-result

	if res.Err != nil {
		span.LogFields(log.String("error", res.Err.Error()))
		ext.Error.Set(span, true)
	}
	return res.Err
}

// addPlayer add the player to the manager instance and make them available for playing games
// player must exist in the userdb
func (m *Manager) addPlayer(in actions.PlayerAction) (*player.Player, error) {
	span, _ := opentracing.StartSpanFromContext(in.Ctx, "addPlayer")
	span.SetTag("playerUsername", in.ClientInfo.PlayerUsername)
	ext.Component.Set(span, "Manager")
	defer span.Finish()

	username := in.ClientInfo.PlayerUsername
	if m.havePlayerUsername(username) {
		return m.getPlayerByUsername(username), actions.ErrUserExists
	}

	m.l.Infof("[%v] Checking for playing in userdb...", username)
	u, err := users.Load(username)
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(
			log.String("error", err.Error()),
		)
		return nil, err
	}

	m.l.Infof("[%v] Adding player to manager...", username)
	p := player.New(u)

	m.players[p.ID] = p
	return p, nil
}

// havePlayerUsername returns true if there is a player with the given username already in the manager
func (m *Manager) havePlayerUsername(username string) bool {
	for _, p := range m.players {
		if p.Username == username {
			return true
		}
	}
	return false
}

// GetPlayerByUsername returns player by username
func (m *Manager) getPlayerByUsername(username string) *player.Player {
	for _, p := range m.players {
		if p.Username == username {
			return p
		}
	}
	return nil
}
