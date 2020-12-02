package player

import (
	"github.com/DanTulovsky/pepper-poker-v2/actions"
	"github.com/DanTulovsky/pepper-poker-v2/poker"
)

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

	// states records how many times a player reach this state
	states map[string]int64
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
		states:      make(map[string]int64),
	}
}

// StateInc increments the state
func (s *Stats) StateInc(state string) {

	if _, ok := s.states[state]; !ok {
		s.states[state] = 0
	}
	s.states[state]++

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
