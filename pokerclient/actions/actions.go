package actions

import (
	"github.com/DanTulovsky/pepper-poker-v2/id"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// PlayerActionResult is the return value of PlayerAction
type PlayerActionResult struct {
	success bool
	err     error
	// arbitrary return value for the caller; is this a good idea?
	rvalue interface{}
}

// NewPlayerActionResult returns the result of a player action
func NewPlayerActionResult(success bool, err error, rvalue interface{}) *PlayerActionResult {
	return &PlayerActionResult{
		success: success,
		err:     err,
		rvalue:  rvalue,
	}
}

// Success returns r.success
func (r *PlayerActionResult) Success() bool {
	return r.success
}

// Error returns r.err
func (r *PlayerActionResult) Error() error {
	return r.err
}

// Rvalue returns r.rvalue
func (r *PlayerActionResult) Rvalue() interface{} {
	return r.rvalue
}

// PlayerAction is sent to the table to act on
type PlayerAction struct {
	PlayerID id.PlayerID
	TableID  id.TableID
	Action   ppb.PlayerAction
	Opts     *ppb.ActionOpts
	Result   chan *PlayerActionResult
}

// NewPlayerAction returns a new PlayerAction
func NewPlayerAction(playerID id.PlayerID, tableID id.TableID, action ppb.PlayerAction, opts *ppb.ActionOpts, resultChan chan *PlayerActionResult) *PlayerAction {
	return &PlayerAction{
		PlayerID: playerID,
		TableID:  tableID,
		Action:   action,
		Opts:     opts,
		Result:   resultChan,
	}
}
