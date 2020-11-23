package actions

import (
	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// PlayerAction encodes what the player can do
type PlayerAction struct {
	Data          *ppb.ClientData
	ToManagerChan chan ManagerAction
}

// NewPlayerAction makes a new playeraction
func NewPlayerAction(in *ppb.ClientData, managerChan chan ManagerAction) PlayerAction {
	return PlayerAction{
		ToManagerChan: managerChan,
		Data:          in,
	}
}

// ManagerAction encodes response back to the player
type ManagerAction struct {
	Result int64
}

// NewManagerAction makes a new ManagerAction
func NewManagerAction(in int64) ManagerAction {
	return ManagerAction{
		Result: in,
	}
}

// TableAction describes a possible table action
type TableAction int

const (
	// TableActionAddPlayer adds a new player to the table
	TableActionAddPlayer TableAction = iota
)

// TableActionRequest is sent to the table
type TableActionRequest struct {
	Action TableAction
}

// NewTableAction returns a table action
func NewTableAction(action TableAction) TableActionRequest {
	return TableActionRequest{
		Action: action,
	}
}

// TableActionResult is the reply to a TableAction
type TableActionResult struct {
}

// NewTableActionResult returns a table action result
func NewTableActionResult() TableActionResult {
	return TableActionResult{}
}
