package poker

import (
	"log"
	"math/rand"

	"github.com/DanTulovsky/deck"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// randomHighCard returns 7 cards with no combination in them
func randomHighCard() []*deck.Card {
	d := deck.NewShuffledDeck()
	cards := []*deck.Card{}

	// Pick 7 random cards until we have no valuable combination
	for {
		// pick random 7 cards
		for i := 0; i < 7; i++ {
			if c, err := d.Next(); err == nil {
				cards = append(cards, c)
			} else {
				log.Fatalf("randomHighCard() error: %v", err)
			}
		}

		hand := BestCombo(cards...)
		if hand.combo == HighCard {
			return cards
		}

		// grab a new deck
		d = deck.NewShuffledDeck()
		cards = nil
	}
}

// randomTwoPair returns 7 cards with at least two pairs in them
func randomTwoPair() []*deck.Card {
	return randomNPair(2)
}

// randomPair returns 7 cards with exactly one pair in them
func randomPair() []*deck.Card {
	return randomNPair(1)
}

// randomNPair returns 7 cards with N pairs in them
func randomNPair(n int) []*deck.Card {
	d := deck.NewShuffledDeck()
	cards := []*deck.Card{}

	uniqueSet := []ppb.CardRank{}
	for i := 0; i < n; i++ {
		p := deck.RandomRankNotIn(uniqueSet...)
		uniqueSet = append(uniqueSet, p)
	}

	for _, p := range uniqueSet {
		s1 := deck.RandomSuit()
		s2 := deck.RandomSuitNotIn(s1)

		cards = append(cards, deck.NewCard(s1, p))
		cards = append(cards, deck.NewCard(s2, p))
		for _, c := range cards {
			d.Remove(c)
		}
	}

	// last random cards
OUTER:
	for len(cards) != 7 {
		var cardsCopy = make([]*deck.Card, len(cards))
		copy(cardsCopy, cards)

		if c, err := d.Next(); err == nil {
			for _, p := range uniqueSet {
				if c.GetRank() == p {
					continue OUTER
				}
			}

			switch n {
			case 2:
				// we don't want three of a kind here, but 3 pairs is ok
				if len(cards) > 4 && c.GetRank() == cards[4].GetRank() {
					continue
				}
				// avoid straights
				if len(cards) > 4 {
					if haveStraight(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveFlush(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveThreeOfAKind(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveFourOfAKind(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveFullHouse(append(cardsCopy, c)) != nil {
						continue OUTER
					}
				}
			case 1:
				// no duplicates in this case
				for _, have := range cards {
					if have.GetRank() == c.GetRank() {
						continue OUTER
					}
				}
				// avoid straights
				if len(cards) > 4 {
					if haveStraight(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveFlush(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveThreeOfAKind(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveFourOfAKind(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveFullHouse(append(cardsCopy, c)) != nil {
						continue OUTER
					}
					if haveTwoPair(append(cardsCopy, c)) != nil {
						continue OUTER
					}
				}
			}

			cards = append(cards, c)
		}
	}

	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// randomNOfAKind returns 7 cards with a N of a kind in them
func randomNOfAKind(n int) []*deck.Card {
	d := deck.NewShuffledDeck()
	cards := []*deck.Card{}

	rank := deck.RandomRank()

	for i := 0; i < n; i++ {
		card := deck.NewCard(ppb.CardSuit(i), rank)
		cards = append(cards, card)
		d.Remove(card)
	}

	// last random cards
	for len(cards) != 7 {
		if c, err := d.Next(); err == nil {
			if c.GetRank() == rank {
				// We don't want four of a kind here
				continue
			}
			cards = append(cards, c)
		}
	}

	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// randomFullHouse returns 7 cards with a full house in them
func randomFullHouse() []*deck.Card {
	d := deck.NewShuffledDeck()
	cards := []*deck.Card{}

	// Pull two different rank cards
	r1, err := d.Next()
	if err != nil {
		log.Fatalf("failed in randomFullHouse(): %v", err)
	}
	cards = append(cards, r1)

	r2, err := d.Next()
	if err != nil {
		log.Fatalf("failed in randomFullHouse(): %v", err)
	}

	// in case r2 was the same rank as r1...
	for r2.IsSameRank(r1) {
		d.Return(r2)
		r2, err = d.Next()
		if err != nil {
			log.Fatalf("failed in randomFullHouse() picking r2: %v", err)
		}
	}
	cards = append(cards, r2)

	byrank := map[ppb.CardRank]int{
		r1.GetRank(): 1,
		r2.GetRank(): 1,
	}

	// pull additional 3 cards of the same rank as r1 or r2
	for len(cards) != 5 {
		next, err := d.Next()
		if err != nil {
			log.Fatalf("failed in randomFullHouse(): %v", err)
		}
		switch {
		case next.GetRank() == r1.GetRank() && byrank[next.GetRank()] < 3:
			cards = append(cards, next)
			byrank[next.GetRank()]++
		case next.GetRank() == r2.GetRank() && byrank[next.GetRank()] < 3:
			cards = append(cards, next)
			byrank[next.GetRank()]++
		}
	}

	// last two random cards
	for i := 0; i < 2; i++ {
		if c, err := d.Next(); err == nil {
			cards = append(cards, c)
		}
	}

	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// randomFlush returns 7 cards with a flush in them
func randomFlush() []*deck.Card {
	d := deck.NewShuffledDeck()

	suit := deck.RandomSuit()
	cards := []*deck.Card{}

	// Pull first 5 cards of the same suit from the deck
	for len(cards) != 5 {
		c, err := d.Next()
		if err != nil {
			log.Fatalf("error making random filush: %v", err)
		}

		// make sure we don't get a straight flush
		if len(cards) == 4 {
			if haveStraight(append(cards, c)) != nil {
				continue
			}

		}
		if c.GetSuit() == suit {
			cards = append(cards, c)
		}
	}

	// last two random cards
	for len(cards) != 7 {
		if c, err := d.Next(); err == nil {
			// make sure we don't make a straight
			if haveStraight(append(cards, c)) != nil {
				continue
			}
			cards = append(cards, c)
		}
	}

	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// randomStraight returns 7 cards with a straight in them
func randomStraight() []*deck.Card {
	// shuffled
	d := deck.NewShuffledDeck()
	cards := []*deck.Card{}

	suit := deck.RandomSuit()
	rank := deck.RandomRankAbove(ppb.CardRank_Five)

	// Start with a random card
	card := deck.NewCard(suit, rank)
	if err := d.Remove(card); err != nil {
		log.Fatalf("error removing card from deck: %v", err)
	}
	cards = append(cards, card)

	// Pull cards fromt he deck until straight is complete
	last := 0
	for len(cards) != 5 {
		next, err := d.Next()
		if err != nil {
			log.Fatalf("error getting random straight: %v", err)
		}

		if next.GetRank() == cards[last].GetRank()-1 {
			cards = append(cards, next)
			last = len(cards) - 1
		} else {
			d.Return(next)
		}
	}

	// last two random cards
	for i := 0; i < 2; i++ {
		if c, err := d.Next(); err == nil {
			cards = append(cards, c)
		}
	}

	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

// randomStraightFlush returns 5 cards with a straight flush in them
func randomStraightFlush() []*deck.Card {
	// shuffled
	d := deck.NewShuffledDeck()
	cards := []*deck.Card{}

	suit := deck.RandomSuit()
	rank := deck.RandomRankAbove(ppb.CardRank_Five)

	// Start with a random card
	card := deck.NewCard(suit, rank)
	if err := d.Remove(card); err != nil {
		log.Fatalf("error removing card from deck: %v", err)
	}
	cards = append(cards, card)

	// Pull cards fromt he deck until straight flush is complete
	last := 0
	for len(cards) != 5 {
		next, err := d.Next()
		if err != nil {
			log.Fatalf("error getting random straight: %v", err)
		}

		if next.GetRank() == cards[last].GetRank()-1 && next.GetSuit() == suit {
			cards = append(cards, next)
			last = len(cards) - 1
		} else {
			d.Return(next)
		}
	}

	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

//  uniqueRankCards returns a list of cards with duplicate ranks removed
func uniqueRankCards(cardSlice []*deck.Card) []*deck.Card {
	keys := make(map[ppb.CardRank]bool)
	list := []*deck.Card{}
	for _, entry := range cardSlice {
		if _, value := keys[entry.GetRank()]; !value {
			keys[entry.GetRank()] = true
			list = append(list, entry)
		}
	}
	return list
}
