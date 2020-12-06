package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

// ActionAddPlayerResult is the result of an AddPlayer action
type ActionAddPlayerResult struct {
	Position int
}

// ActionInfoResult is the result of an ActioInfo action
type ActionInfoResult struct {
	AvailableToJoin        bool
	Name                   string
	MaxPlayers, MinPlayers int
}

// ActionRequest is sent to the table
type ActionRequest struct {
	Action actions.TableAction
	Player *player.Player
	Opts   interface{}

	resultChan chan ActionResult
}

// NewTableAction returns a table action
func NewTableAction(action actions.TableAction, ch chan ActionResult, p *player.Player, opts interface{}) ActionRequest {
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
