package player

import (
	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// Money keeps track of the player's money during the hand
type Money struct {
	bank, stack, betThisRound, betThisHand, winnings int64
}

// NewMoney returns a new money struct
func NewMoney(bank int64) *Money {
	return &Money{
		bank: bank,
	}
}

// Bank returns the player's total money
func (pm *Money) Bank() int64 {
	return pm.bank
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

// AsProto returns this as a proto
func (pm *Money) AsProto() *ppb.PlayerMoney {
	return &ppb.PlayerMoney{
		Bank:         pm.bank,
		Stack:        pm.stack,
		BetThisRound: pm.betThisRound,
		BetThisHand:  pm.betThisHand,
		Winnings:     pm.winnings,
	}
}
