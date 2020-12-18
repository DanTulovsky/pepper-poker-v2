package player

import (
	"time"

	"github.com/DanTulovsky/deck"
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/id"
	"github.com/DanTulovsky/pepper-poker-v2/poker"
	"github.com/DanTulovsky/pepper-poker-v2/server/users"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	playerCombos = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pepperpoker_combos_played_total",
		Help: "The total number of combos played",
	}, []string{"username", "combo"}) // TODO: this is only ok for very few players

	playerActions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pepperpoker_player_actions_total",
		Help: "The total number of player actions",
	}, []string{"username", "action"}) // TODO: This is only ok for very few players

	playerMoney = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pepperpoker_player_money_total",
		Help: "Player money stats",
	}, []string{"username", "stat"}) // TODO: This is only ok for very few players

	playerStates = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pepperpoker_player_states_total",
		Help: "The total number of states reached by player",
	}, []string{"username", "state"}) // TODO: This is only ok for very few players
)

// handInfo is info for each hand (one poker game)
type handInfo struct {
	folded bool
	allin  bool

	// Keep track if action is required
	actionRequired bool

	Hole []deck.Card
	Hand *poker.PlayerHand
}

// newHandInfo creates a new hand info
func newHandInfo() *handInfo {
	return &handInfo{
		folded:         false,
		allin:          false,
		actionRequired: false,
		Hole:           []deck.Card{},
	}
}

// LastAction describes the last action of the player
type LastAction struct {
	Action ppb.PlayerAction
	Amount int64
}

// Player represents a single player
type Player struct {
	ID       id.PlayerID
	Name     string
	Username string

	// Keeps track of how many turns the player took, used to sync the client
	CurrentTurn int64
	HandInfo    *handInfo

	LastAction    LastAction
	TablePosition int
	WaitSince     time.Time // time the player becam the active player

	money    *Money
	iswinner bool

	Stats *Stats

	CommChannel chan actions.GameData
}

// New creates a new player
func New(u users.User) *Player {
	return &Player{
		ID:            id.NewPlayerID(),
		Name:          u.Name,
		Username:      u.Username,
		money:         NewMoney(u.Bank),
		HandInfo:      newHandInfo(),
		Stats:         NewStats(u.Username),
		TablePosition: -1,
	}
}

// String returns ...
func (p *Player) String() string {
	return p.Username
}

// InList returns true if the player is in the list l
func (p *Player) InList(l []*Player) bool {
	for _, pl := range l {
		if p == pl {
			return true
		}
	}
	return false
}

// DisconnectReset is called after player disconnects
func (p *Player) DisconnectReset() {
	p.HandInfo = newHandInfo()
	if p.CommChannel != nil {
		close(p.CommChannel)
		p.CommChannel = nil
	}
	p.SetLastAction(actions.TableAction(ppb.PlayerAction_PlayerActionNone), 0)
}

// ResetForBettingRound resets the player for the next betting round (multiple of these inside one hand)
func (p *Player) ResetForBettingRound() {
	p.Money().SetBetThisRound(0)
	p.Money().SetWinnings(0)
}

// PlayerHand sets the player's final hand
func (p *Player) PlayerHand() *poker.PlayerHand {
	return p.HandInfo.Hand
}

// SetPlayerHand sets the player's final hand
func (p *Player) SetPlayerHand(h *poker.PlayerHand) {
	p.HandInfo.Hand = h
}

// IsWinner retruns true if the player is a winner
func (p *Player) IsWinner() bool {
	return p.iswinner
}

// SetWinnerAndWinnings sets the player as a winner, and adjusts the money
func (p *Player) SetWinnerAndWinnings(w int64) {
	p.iswinner = true
	p.money.SetWinnings(w)
}

// AsProto returns the player as an proto
// No confidential information is returned here
func (p *Player) AsProto(bigBlind, buyin int64) *ppb.Player {
	s := &ppb.Player{
		Name:     p.Name,
		Id:       p.ID.String(),
		Position: int64(p.TablePosition),
		Money:    p.money.AsProto(),
		State:    []ppb.PlayerState{ppb.PlayerState_PlayerStateDefault},
		LastAction: &ppb.LastAction{
			Action: p.LastAction.Action,
			Amount: p.LastAction.Amount,
		},
	}

	if p.Folded() {
		s.State = append(s.State, ppb.PlayerState_PlayerStateFolded)
	}

	if p.Money().Stack() < bigBlind {
		s.State = append(s.State, ppb.PlayerState_PlayerStateStackEmpty)

		if p.Money().Bank() < buyin {
			s.State = append(s.State, ppb.PlayerState_PlayerStateBankEmpty)
		}
	}
	return s
}

// Money returns the player's money
func (p *Player) Money() *Money {
	return p.money
}

// SetLastAction sets the last action done by the player
func (p *Player) SetLastAction(a actions.TableAction, amount int64) {
	la := LastAction{}

	switch a {
	case actions.ActionCall:
		la.Action = ppb.PlayerAction_PlayerActionCall
	case actions.ActionBet:
		la.Action = ppb.PlayerAction_PlayerActionBet
		la.Amount = amount
	case actions.ActionCheck:
		la.Action = ppb.PlayerAction_PlayerActionCheck
	case actions.ActionFold:
		la.Action = ppb.PlayerAction_PlayerActionFold
	case actions.ActionAllIn:
		la.Action = ppb.PlayerAction_PlayerActionAllIn
		la.Amount = amount
	}

	p.LastAction = la
	p.Stats.ActionInc(a)
}

// AddHoleCard deals adds a card to the player's hole
func (p *Player) AddHoleCard(c deck.Card) {
	p.HandInfo.Hole = append(p.HandInfo.Hole, c)
}

// Hole returns the player's hole
func (p *Player) Hole() []deck.Card {
	return p.HandInfo.Hole
}

// Init initializes the player to play a single hand (one poker game)
func (p *Player) Init() {
	p.iswinner = false
	p.HandInfo = newHandInfo()
	p.SetLastAction(actions.TableAction(ppb.PlayerAction_PlayerActionNone), 0)
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

// GoAllIn marks the player as having gone all in if true
func (p *Player) GoAllIn(g bool) {
	p.HandInfo.allin = g
}
