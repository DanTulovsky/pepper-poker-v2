package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

// Action describes a possible table action
type Action int

const (
	// ActionAddPlayer adds a new player to the table
	ActionAddPlayer Action = iota
)

// ActionAddPlayerResult is the result of an AddPlayer action
type ActionAddPlayerResult struct {
	Position int
}

// ActionRequest is sent to the table
type ActionRequest struct {
	Action Action
	Player *player.Player
	Opts   interface{}

	resultChan chan ActionResult
}

// NewTableAction returns a table action
func NewTableAction(action Action, ch chan ActionResult, p *player.Player, opts interface{}) ActionRequest {
	return ActionRequest{
		Action:     action,
		Player:     p,
		Opts:       opts,
		resultChan: ch,
	}
}

// ActionResult is the reply to a TableAction
type ActionResult struct {
	Err    error
	Result interface{}
}

// NewTableActionResult returns a table action result
func NewTableActionResult(err error, r interface{}) ActionResult {
	return ActionResult{
		Err:    err,
		Result: r,
	}
}