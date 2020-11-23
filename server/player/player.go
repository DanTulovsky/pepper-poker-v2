package player

import (
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
)

// Player represents a single player
type Player struct {
	ID   id.PlayerID
	Name string

	CommChannel chan actions.ManagerAction
}

// New creates a new player
func New(name string, cc chan actions.ManagerAction) *Player {
	return &Player{
		ID:          id.NewPlayerID(),
		Name:        name,
		CommChannel: cc,
	}
}
