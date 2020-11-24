package actions

import (
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
}

// NewPlayerAction makes a new playeraction
func NewPlayerAction(action ppb.PlayerAction, opts *ppb.ActionOpts, ci *ppb.ClientInfo, managerChan chan GameData, resultc chan PlayerActionResult) PlayerAction {
	return PlayerAction{
		Action:       action,
		Opts:         opts,
		ClientInfo:   ci,
		ToClientChan: managerChan,
		ResultC:      resultc,
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
