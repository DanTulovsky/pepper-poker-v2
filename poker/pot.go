package poker

import (
	"fmt"
	"log"

	"github.com/DanTulovsky/pepper-poker-v2/id"
)

// Subpot is one subpot within the pot.
type Subpot struct {
	limit int64
	bets  map[id.PlayerID]int64
}

// NewSubpot creates a new subpot.
func NewSubpot() *Subpot {
	return &Subpot{
		bets: make(map[id.PlayerID]int64),
	}
}

// GetTotal returns the total amount of money in the subpot.
func (s *Subpot) GetTotal() int64 {
	total := int64(0)
	for _, b := range s.bets {
		total += b
	}
	return total
}

// GetBet returns a player's total bet in the subpot.
func (s *Subpot) GetBet(player id.PlayerID) int64 {
	return s.bets[player]
}

// Pot contains all the information about the pots.
type Pot struct {
	subpots []*Subpot

	// Only set after Finalize is called
	finalized bool
	winnings  map[id.PlayerID]int64
}

// NewPot creates a new pot.
func NewPot() *Pot {
	p := &Pot{
		subpots:  []*Subpot{NewSubpot()},
		winnings: make(map[id.PlayerID]int64),
	}
	return p
}

// GetTotal returns the total amount of money in the pot.
func (p *Pot) GetTotal() int64 {
	total := int64(0)
	for _, s := range p.subpots {
		total += s.GetTotal()
	}
	return total
}

// GetBet returns a player's total bet in the pot.
func (p *Pot) GetBet(player id.PlayerID) int64 {
	bet := int64(0)
	for _, s := range p.subpots {
		bet += s.GetBet(player)
	}
	return bet
}

// Add adds a player's bet to the pot.
func (p *Pot) Add(player id.PlayerID, bet int64, allin bool) {
	if bet <= 0 {
		log.Fatal("trying to add non-positive bet to the pot")
	}
	if p.finalized {
		p.finalized = false
		p.winnings = make(map[id.PlayerID]int64)
	}

	// Go through the subpots, adding the player's bet to the first subpots we can.
	last := -1
	for i, s := range p.subpots {
		// Calculate how much the player can put into this subpot.
		diff := bet
		if s.limit != 0 && s.limit-s.bets[player] < diff {
			diff = s.limit - s.bets[player]
		}
		if diff == 0 {
			continue
		}

		// Add this amount to the subpot.
		s.bets[player] += diff
		bet -= diff

		// If there's no remaining money, this is the last subpot.
		if bet == 0 {
			last = i
			break
		}
	}
	// If we've gone through all the subpots but we haven't managed to add this entire bet, we need to add a new subpot.
	if bet != 0 {
		p.subpots = append(p.subpots, NewSubpot())
		last = len(p.subpots) - 1
		p.subpots[last].bets[player] += bet
		bet = 0
	}

	// Finally, if the player just went allin, we may need to split the last subpot in the case where the player's
	// bet in that subpot is less than other players'.
	if allin {
		// The subpot's new limit is the player's bet in that pot. Create a new subpot whose limit is the overflow.
		limit := p.subpots[last].bets[player]
		s := NewSubpot()
		if p.subpots[last].limit != 0 {
			s.limit = p.subpots[last].limit - limit
		}
		p.subpots[last].limit = limit

		// Move the overflow of any other players' bets that are over the new limit into the new subpot.
		for k, v := range p.subpots[last].bets {
			diff := v - p.subpots[last].limit
			if diff > 0 {
				p.subpots[last].bets[k] -= diff
				s.bets[k] += diff
			}
		}

		// If the new subpot is not empty, splice it into the list of subpots.
		if len(s.bets) != 0 {
			p.subpots = append(p.subpots, s)
			copy(p.subpots[last+2:], p.subpots[last+1:])
			p.subpots[last+1] = s
		}
	}
}

// Finalize finalizes each player's winnings based on their hand rankings.
func (p *Pot) Finalize(rankings []Winners) {
	for _, s := range p.subpots {
		if s.GetTotal() == 0 {
			continue
		}
		// Each subpot can only go to players who have money in the subpot.
		var winners []id.PlayerID
		for _, level := range rankings {
			// Add all players at this level who have money in this subpot.
			for _, player := range level {
				if _, ok := s.bets[player]; ok {
					winners = append(winners, player)
				}
			}
			// If we found any winners, this is the winning level.
			if len(winners) > 0 {
				break
			}
		}
		// Unclaimed money shouldn't really happen (unless maybe a player leaves in the middle of a hand).
		if len(winners) == 0 {
			log.Printf("unclaimed money in subpot: %v", s)
			continue
		}
		// Divide the subpot among the winners, awarding leftovers in order (hopefully, clockwise after button).
		winning := s.GetTotal() / int64(len(winners))
		remainder := s.GetTotal() - (winning * int64(len(winners)))
		for _, winner := range winners {
			p.winnings[winner] += winning
			if remainder > 0 {
				p.winnings[winner]++
				remainder--
			}
		}
	}
	p.finalized = true
}

// GetWinnings returns a player's winnings after the pot has been finalized.
func (p *Pot) GetWinnings(player id.PlayerID) (int64, error) {
	if !p.finalized {
		return 0, fmt.Errorf("must finalize the pot before getting winnings")
	}
	return p.winnings[player], nil
}
