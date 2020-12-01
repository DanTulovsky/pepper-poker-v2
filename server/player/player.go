package player

import (
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

// Stats keeps per-player stats
type Stats struct {
	// username for metric exporting
	username string

	// GamesPlays is the total number of games played
	gamesPlayed int64

	// GamesWon is the number of games won (no money lost; in the winners list)
	gamesWon int64

	// Combos is a map of combo to how many times player had it
	combos map[poker.Combo]int64

	// Actions is a map of action to how many times the player acted
	actions map[actions.Action]int64

	// TODO: Add money related stats
	money map[string]int64
}

// GamesPlayedInc increments games played
func (s *Stats) GamesPlayedInc() {
	s.gamesPlayed++
}

// NewStats returns new stats
func NewStats(username string) *Stats {
	return &Stats{
		username:    username,
		gamesPlayed: 0,
		gamesWon:    0,
		combos:      make(map[poker.Combo]int64),
		actions:     make(map[actions.Action]int64),
		money:       make(map[string]int64),
	}
}

// StateInc increments the state
func (s *Stats) StateInc(state string) {

	playerStates.WithLabelValues(s.username, state).Inc()
}

// MoneySet sets the money stat
func (s *Stats) MoneySet(stat string, amount int64) {

	switch stat {
	case "winnings":
		if _, ok := s.money[stat]; !ok {
			s.money[stat] = 0
		}
		s.money[stat] += amount
	}

	playerMoney.WithLabelValues(s.username, stat).Set(float64(amount))
}

// GamesWonInc increments games won
func (s *Stats) GamesWonInc() {
	s.gamesWon++
}

// ComboInc increments the combo count
func (s *Stats) ComboInc(combo poker.Combo) {

	if _, ok := s.combos[combo]; !ok {
		s.combos[combo] = 0
	}
	s.combos[combo]++

	playerCombos.WithLabelValues(s.username, combo.String()).Inc()
}

// ActionInc increments the action count
func (s *Stats) ActionInc(a actions.Action) {

	if _, ok := s.actions[a]; !ok {
		s.actions[a] = 0
	}
	s.actions[a]++

	playerActions.WithLabelValues(s.username, a.String()).Inc()
}

// Player represents a single player
type Player struct {
	ID       id.PlayerID
	Name     string
	Username string

	// Keeps track of how many turns the player took, used to sync the client
	CurrentTurn   int64
	HandInfo      *handInfo
	TablePosition int

	money    *Money
	iswinner bool

	Stats *Stats

	CommChannel chan actions.GameData
}

// New creates a new player
func New(u users.User) *Player {
	return &Player{
		ID:       id.NewPlayerID(),
		Name:     u.Name,
		Username: u.Username,
		money:    NewMoney(u.Bank),
		HandInfo: newHandInfo(),
		Stats:    NewStats(u.Username),
	}
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

// ResetForBettingRound resets the player for the next betting round (multiple of these inside one hand)
func (p *Player) ResetForBettingRound() {
	p.Money().SetBetThisRound(0)
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
func (p *Player) AsProto() *ppb.Player {
	return &ppb.Player{
		Name:     p.Name,
		Id:       p.ID.String(),
		Position: int64(p.TablePosition),
		Money:    p.money.AsProto(),
	}
}

// Money returns the player's money
func (p *Player) Money() *Money {
	return p.money
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
