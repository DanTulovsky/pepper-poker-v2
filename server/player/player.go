package player

import (
	"github.com/DanTulovsky/deck"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/poker"
)

// handInfo is info for each hand (one poker game)
type handInfo struct {
	folded bool
	allin  bool

	// Keep track if action is required
	actionRequired bool

	Cards []*deck.Card
	Hand  *poker.Hand
}

// newHandInfo creates a new hand info
func newHandInfo() *handInfo {
	return &handInfo{
		folded:         false,
		allin:          false,
		actionRequired: false,
	}
}

// Player represents a single player
type Player struct {
	ID   id.PlayerID
	Name string

	// Keeps track of how many turns the player took, used to sync the client
	CurrentTurn int64
	HandInfo    *handInfo

	Money *Money

	CommChannel chan actions.GameData
}

// New creates a new player
func New(name string) *Player {
	return &Player{
		ID:   id.NewPlayerID(),
		Name: name,
	}
}

// InitHand initializes the player to play a single hand (one poker game)
func (p *Player) InitHand() {
	p.HandInfo = newHandInfo()
}

// SetActionRequired sets action required
func (p *Player) SetActionRequired(a bool) {
	p.HandInfo.actionRequired = a
}

// ActionRequired returns true if player action is required
func (p *Player) ActionRequired() bool {
	switch {
	case p.HandInfo.allin:
		return false
	case p.HandInfo.folded:
		return false
	default:
		return p.HandInfo.actionRequired
	}
}

// Folded returns the fold state of the player
func (p *Player) Folded() bool {
	return p.HandInfo.folded
}

// Fold marks the player as folded
func (p *Player) Fold() {
	p.HandInfo.folded = true
}

// AllIn returns the allin state of the player
func (p *Player) AllIn() bool {
	return p.HandInfo.allin
}

// GoAllIn marks the player as having gone all in
func (p *Player) GoAllIn() {
	p.HandInfo.allin = true
}
