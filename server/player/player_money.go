package player

// Money keeps track of the player's money during the hand
type Money struct {
	stack, betThisRound, winnings int64
}

// Stack returns the player's stack (or what's left of it)
func (pm *Money) Stack() int64 {
	return pm.stack
}

// BetThisRound returns the amount of money bet during the current betting round
func (pm *Money) BetThisRound() int64 {
	return pm.betThisRound
}

// Winnings returns the amount won at the end of the hand
func (pm *Money) Winnings() int64 {
	return pm.winnings
}

// SetWinnings sets the amount won at the end of the hand
func (pm *Money) SetWinnings(winnings int64) {
	pm.winnings = winnings
}
