package actions

import (
	"context"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// PlayerAction is sent on Play request
type PlayerAction struct {
	ClientInfo *ppb.ClientInfo
	Action     ppb.PlayerAction
	Opts       *ppb.ActionOpts

	// use this channel to send back game data to the client
	ToClientChan chan GameData

	// use this channel to send back and error to the grpc server on the initial subscription
	ResultC chan PlayerActionResult

	// Pass RPC context into the Manager (used in tracing)
	Ctx context.Context
}

// NewPlayerAction makes a new playeraction
func NewPlayerAction(ctx context.Context, action ppb.PlayerAction, opts *ppb.ActionOpts, ci *ppb.ClientInfo, managerChan chan GameData, resultc chan PlayerActionResult) PlayerAction {
	return PlayerAction{
		Action:       action,
		Opts:         opts,
		ClientInfo:   ci,
		ToClientChan: managerChan,
		ResultC:      resultc,
		Ctx:          ctx,
	}
}

// PlayerActionResult is the result from the manager to the grpc server
type PlayerActionResult struct {
	Result interface{}
	Err    error
}

// NewPlayerActionResult makes a new PlayerAction
func NewPlayerActionResult(err error, result interface{}) PlayerActionResult {
	return PlayerActionResult{
		Result: result,
		Err:    err,
	}
}

// NewPlayerActionError returns an error for a playeraction
func NewPlayerActionError(err error) PlayerActionResult {
	return PlayerActionResult{
		Err: err,
	}
}

// GameData encodes response back to the player
type GameData struct {
	Data *ppb.GameData
}

// NewGameData makes a new ManagerAction
func NewGameData(in *ppb.GameData) GameData {
	return GameData{
		Data: in,
	}
}

// TableAction describes a possible table action
type TableAction int

func (a TableAction) String() string {
	switch a {
	case ActionNone:
		return ""
	case ActionBet:
		return "Bet"
	case ActionBuyIn:
		return "Buyin"
	case ActionCheck:
		return "Check"
	case ActionCall:
		return "Call"
	case ActionFold:
		return "Fold"
	case ActionDisconnect:
		return "Disconnect"
	}
	return ""
}

const (
	// ActionNone is the default
	ActionNone TableAction = iota

	// ActionAddPlayer adds a new player to the table
	ActionAddPlayer

	// ActionRegisterPlayerCC adds the player comm channel for sending updates
	ActionRegisterPlayerCC

	// ActionInfo returns various information about the table
	ActionInfo

	// ActionAckToken acks a token
	ActionAckToken

	// ActionCheck checks
	ActionCheck

	// ActionCall calls
	ActionCall

	// ActionFold folds
	ActionFold

	// ActionBet bets
	ActionBet

	// ActionAllIn bets all available money
	ActionAllIn

	// ActionBuyIn uses the player's bank to buy into the table (bank -> stack)
	ActionBuyIn

	// ActionDisconnect is triggered on client disconnect
	ActionDisconnect
)
