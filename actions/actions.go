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
