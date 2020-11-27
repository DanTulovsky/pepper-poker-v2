package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

// Action describes a possible table action
type Action int

const (
	// ActionAddPlayer adds a new player to the table
	ActionAddPlayer Action = iota

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
