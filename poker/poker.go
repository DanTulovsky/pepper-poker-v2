package poker

import (
	"fmt"
	"log"
	"sort"

	"github.com/DanTulovsky/deck"
	"github.com/DanTulovsky/pepper-poker-v2/id"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

// Combo is a combination of 5 cards within a set of 7 cards
type Combo int

const (
	// Unknown combination
	Unknown Combo = iota
	// HighCard is a high card
	HighCard
	// Pair is one pair
	Pair
	// TwoPair is two pairs
	TwoPair
	// ThreeOfAKind is three of a kind
	ThreeOfAKind
	// Straight is a straight
	Straight
	// Flush is a flush
	Flush
	// FullHouse is a full house
	FullHouse
	// FourOfAKind is four of a kind
	FourOfAKind
	// StraightFlush is a straight flush
	StraightFlush
	// RoyalFlush is a special case of a straight flush
	RoyalFlush
)

var (
	// ComboToString ...
	ComboToString = map[Combo]string{
		Unknown:       "unknown",
		HighCard:      "High Card",
		Pair:          "Pair",
		TwoPair:       "Two Pair",
		ThreeOfAKind:  "Three of a Kind",
		Straight:      "Straight",
		Flush:         "Flush",
		FullHouse:     "Full House",
		FourOfAKind:   "Four of a Kind",
		StraightFlush: "Straight Flush",
	}

	nofakind = map[int]Combo{
		2: Pair,
		3: ThreeOfAKind,
		4: FourOfAKind,
	}
)

// Hand is a poker hand of five cards
type Hand struct {
	cards []deck.Card
	combo Combo
}

// NewHand returns a new hand
func NewHand() *Hand {
	return &Hand{
		cards: []deck.Card{},
	}
}

// NewHandFrom returns a new hand from provided input
func NewHandFrom(cards []deck.Card, combo Combo) *Hand {
	return &Hand{
		cards: cards,
		combo: combo,
	}
}

// Cards return h.cards
func (h *Hand) Cards() []deck.Card {
	return h.cards
}

// Combo returns h.combo
func (h *Hand) Combo() Combo {
	return h.combo
}
func (h *Hand) String() string {
	return fmt.Sprintf("(%v) -> [%v]", ComboToString[h.combo], h.cards)
}

// SortCards sorts the cards, highest to lowest, in the hand based on the combination
// Allows easy comparison of hands with the same combo
func (h *Hand) SortCards() {

	switch h.combo {
	case HighCard:
		sort.Sort(sort.Reverse(deck.SortByCards(h.cards)))
	case Pair:
		h.cards = SortPair(h.cards)
	case TwoPair:
		h.cards = SortTwoPair(h.cards)
	case ThreeOfAKind:
		h.cards = SortThreeOfAKind(h.cards)
	case Straight:
		h.cards = SortStraight(h.cards)
	case Flush:
		h.cards = SortFlush(h.cards)
	case FullHouse:
		h.cards = SortFullHouse(h.cards)
	case FourOfAKind:
		h.cards = SortFourOfAKind(h.cards)
	case StraightFlush:
		h.cards = SortStraightFlush(h.cards)
	}
}

// CompareTo returns -1 if h < other; 0 if h == other; 1 if h > other
func (h *Hand) CompareTo(other *Hand) int {

	switch {
	case h.combo < other.combo:
		return -1
	case h.combo > other.combo:
		return 1
	}

	// combo is the same, sort if needed
	// note that most combinations are pre-sorted
	h.SortCards()
	other.SortCards()

	return CompareCards(h, other)
}

// PlayerHand encapsulates the 7 cards of each player and the playerID
type PlayerHand struct {
	Cards []deck.Card
	Hand  *Hand
	ID    id.PlayerID
}

// NewPlayerHand returns a new player hand
func NewPlayerHand(id id.PlayerID, cards []deck.Card) *PlayerHand {
	return &PlayerHand{
		ID:    id,
		Cards: cards,
	}
}

// SortCards calls the Hand.SortCards() function
func (pl *PlayerHand) SortCards() {
	pl.Hand.SortCards()
}

// SortByPlayerHands sorts by player hands
type SortByPlayerHands []*PlayerHand

func (a SortByPlayerHands) Len() int      { return len(a) }
func (a SortByPlayerHands) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortByPlayerHands) Less(i, j int) bool {
	comp := a[i].Hand.CompareTo(a[j].Hand)
	switch comp {
	case -1:
		return true
	case 0:
		return true
	default: // 1
		return false
	}
}

// Winners is a list of player IDs that have the same value hands
type Winners []id.PlayerID

// SortByID sorts the winners
type SortByID []id.PlayerID

func (a SortByID) Len() int      { return len(a) }
func (a SortByID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortByID) Less(i, j int) bool {
	return a[i] < a[j]
}

// BestHand returns an ordered list of lists of the player IDs of the best hands (in case of a tie)
// [
//	[player1],
//  [player4, player7],
//  [player2]
// ]
// Only players that have not folded are included
func BestHand(pls []*PlayerHand) []Winners {

	winners := []Winners{}

	// First get the best hand for all players that haven't folded (those passed in here)
	for _, p := range pls {
		h := BestCombo(p.Cards...)
		h.SortCards()
		p.Hand = h // save hand for each player
	}

	sort.Sort(sort.Reverse(SortByPlayerHands(pls)))

	first := 0

	for first < len(pls)-1 {
		// The current "level" of winners"
		win := Winners{}
		used := 0

		// first definitely wins
		win = append(win, pls[first].ID)
		used++

		for _, ph := range pls[first+1:] {
			switch pls[first].Hand.CompareTo(ph.Hand) {
			case 0: // same as the previous player
				win = append(win, ph.ID)
				used++
			default:
				break
			}
		}

		first += used
		winners = append(winners, win)
	}

	// there might be one last one left
	if first < len(pls) {
		if first+1 != len(pls) {
			log.Fatalf("there should only be one left here, have: %v", len(pls)-first)
		}

		win := Winners{
			pls[first].ID,
		}

		winners = append(winners, win)
	}

	// consistent order for comparisons
	for _, w := range winners {
		sort.Sort(SortByID(w))
	}
	return winners
}

// BestCombo returns the best hand out of the seven cards given
func BestCombo(cards ...deck.Card) *Hand {
	var hand *Hand

	fourOfAKind := haveFourOfAKind(cards)
	fullHouse := haveFullHouse(cards)
	flush := haveFlush(cards)
	straight := haveStraight(cards)

	// must check after flush and straight
	var straightFlush *Hand
	if flush != nil && straight != nil {
		straightFlush = haveStraightFlush(flush.cards, cards)
	}

	threeOfAKind := haveThreeOfAKind(cards)
	twoPair := haveTwoPair(cards)
	pair := havePair(cards)
	highCard := haveHighCard(cards)

	// first non-nil is the best one
	handList := []*Hand{straightFlush, fourOfAKind, fullHouse, flush, straight, threeOfAKind, twoPair, pair, highCard}
	for _, h := range handList {
		if h != nil {
			return h
		}
	}

	// It should not be possible to get here, hard fail
	log.Fatalf("hand did not match any combinations (impossible): %v", hand)
	return nil
}

// haveStraightFlash returns *Hand that makes up the straight flush, or nil
func haveStraightFlush(flush, cards []deck.Card) *Hand {
	if cards == nil || flush == nil || len(cards) == 0 || len(flush) == 0 {
		return nil
	}

	// get the suite of the flush
	suit := flush[0].Suite

	// get all the cards of this suit in cards
	allsuitcards := []deck.Card{}
	for _, c := range cards {
		if c.Suite == suit {
			allsuitcards = append(allsuitcards, c)
		}
	}

	// check if there is a straight in allsuitcards
	straight := haveStraight(allsuitcards)
	if straight != nil {
		straight.combo = StraightFlush
	}
	return straight
}

type rankCards struct {
	r ppb.CardRank
	c int
}

func (l rankCards) String() string {
	return fmt.Sprintf("%v (%v)", ppb.CardRank(l.r), l.c)
}

// haveFullHouse returns *Hand that makes up the full house, or nil
func haveFullHouse(cards []deck.Card) *Hand {
	if len(cards) < 5 {
		log.Fatalf("need at least 5 cards for haveFullHouse, have %d", len(cards))
	}

	var hand = new(Hand)

	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	byrank := deck.CountByRank(cards)

	highest := rankCards{}
	secondHeighest := rankCards{}

	// find the highest triplet of cards
	for rank, count := range byrank {
		if count < 3 {
			continue
		}
		if count >= highest.c && rank >= highest.r {
			highest.r = rank
			highest.c = count
		}
	}
	delete(byrank, highest.r)

	// find the highest tuple
	for rank, count := range byrank {
		if count < 2 {
			continue
		}
		if count >= secondHeighest.c && rank >= secondHeighest.r {
			secondHeighest.r = rank
			secondHeighest.c = count
		}
	}
	// log.Printf("highest: %v", highest)
	// log.Printf("secondHighest: %v", secondHeighest)

	// need at lest 3 of highest and at least 2 of second highest
	if highest.c > 2 && secondHeighest.c > 1 {
		// add up to 3 of the highest cards
		for _, c := range cards {
			if c.GetRank() == highest.r {
				hand.cards = append(hand.cards, c)
				if len(hand.cards) == 3 {
					break
				}
			}
		}

		// and up to 3 of the lower cards to a total of 5
		for _, c := range cards {
			if c.GetRank() == secondHeighest.r {
				hand.cards = append(hand.cards, c)
				if len(hand.cards) == 5 {
					break
				}
			}
		}
		hand.combo = FullHouse
		return hand
	}

	return nil
}

// haveStraight returns *Hand that makes up the straight, or nil
func haveStraight(cards []deck.Card) *Hand {
	if len(cards) < 5 {
		log.Fatalf("need at least 5 cards for haveStraight, have %d", len(cards))
	}

	var hand = new(Hand)

	cards = uniqueRankCards(cards)
	if len(cards) < 5 {
		return nil
	}
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	var straight []deck.Card

	// If the first 3 cards are consecutive, straight starts from the first card
	if cards[0].GetRank() == cards[1].GetRank()+1 && cards[0].GetRank() == cards[2].GetRank()+2 {
		// check for straight from first card
		straight = append(straight, cards[0])
		straight = append(straight, cards[1])
		straight = append(straight, cards[2])

		next := 3

		for i := 3; i < len(cards); i++ {
			if cards[0].GetRank() == cards[i].GetRank()+ppb.CardRank(next) {
				straight = append(straight, cards[i])
				next++
			}
			if len(straight) == 5 {
				hand.cards = straight
				hand.combo = Straight
				return hand
			}
		}
		return nil
	}

	straight = nil
	// If the second and third cards are consecutive, straight starts from the second card
	if cards[1].GetRank() == cards[2].GetRank()+1 {
		// check for straight from the second card
		straight = append(straight, cards[1])
		straight = append(straight, cards[2])

		next := 2

		for i := 3; i < len(cards); i++ {
			if cards[1].GetRank() == cards[i].GetRank()+ppb.CardRank(next) {
				straight = append(straight, cards[i])
				next++
			}
			if len(straight) == 5 {
				hand.cards = straight
				hand.combo = Straight
				return hand
			}
		}

		// Reset hand and fall through, as we might have
		// A, Q, J, 2, 3, 4, 5
		hand = nil
	}

	straight = nil
	// Or straight stats from the 3rd card
	straight = append(straight, cards[2])
	next := 1

	for i := 3; i < len(cards); i++ {
		if cards[2].GetRank() == cards[i].GetRank()+ppb.CardRank(next) {
			straight = append(straight, cards[i])
			next++
		}
		if len(straight) == 5 {
			hand.cards = straight
			hand.combo = Straight
			return hand
		}
	}

	return haveAceLowStraight(cards)
}

func haveAceLowStraight(cards []deck.Card) *Hand {

	var hand = new(Hand)

	// check for Ace low straight, all of these ranks are required
	ranks := []ppb.CardRank{ppb.CardRank_Ace, ppb.CardRank_Two, ppb.CardRank_Three, ppb.CardRank_Four, ppb.CardRank_Five}
	for _, r := range ranks {
		if !deck.RankInList(r, cards) {
			// missing requied rank
			return nil
		}
	}

	// have all required ranks, find them
	byrank := deck.CardsByRank(cards)
	for _, r := range ranks {
		hand.cards = append(hand.cards, byrank[r][0])
		hand.combo = Straight
	}

	return hand
}

// haveFlush returns *Hand that makes up the flush, or nil
func haveFlush(cards []deck.Card) *Hand {

	if len(cards) < 5 {
		log.Fatalf("need at least 5 cards for haveFlush, have %d", len(cards))
	}

	var hand = new(Hand)
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	// count the number of each suit
	bysuit := deck.CountBySuit(cards)

	for suit, count := range bysuit {
		if count >= 5 {
			for _, card := range cards {
				if card.GetSuit() == suit {
					hand.cards = append(hand.cards, card)
				}

				if len(hand.cards) == 5 {
					hand.combo = Flush
					return hand
				}
			}
		}
	}

	return nil
}

func haveFourOfAKind(cards []deck.Card) *Hand {
	return haveNOfAKind(cards, 4)
}

func haveThreeOfAKind(cards []deck.Card) *Hand {
	return haveNOfAKind(cards, 3)
}

// haveNOfAKind returns *Hand that makes up N of a kind, or nil
func haveNOfAKind(cards []deck.Card, n int) *Hand {

	if len(cards) < 5 {
		log.Fatalf("need at least 5 cards for haveNOfAKnd, have %d", len(cards))
	}

	var hand = new(Hand)
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	byrank := deck.CardsByRank(cards)

	for _, cardList := range byrank {
		if len(cardList) == n {
			for _, c := range cardList {
				hand.cards = append(hand.cards, c)
			}
			hand.combo = nofakind[n]

			// Add 7-n more cards
			for _, c := range cards {
				if !deck.CardInList(c, hand.cards) {
					hand.cards = append(hand.cards, c)
				}
				if len(hand.cards) == 5 {
					return hand
				}
			}
		}
	}

	return nil
}

// haveTwoPair returns *Hand that makes up two pair, or nil
func haveTwoPair(cards []deck.Card) *Hand {

	if len(cards) < 5 {
		log.Fatalf("need at least 5 cards for haveTwoPair, have %d", len(cards))
	}

	var hand = new(Hand)
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	byrank := deck.CardsByRank(cards)

	// possible pairs, take the first 4 cards here at the end
	possible := []deck.Card{}
	for _, cardList := range byrank {
		// HaveThreeOfAKind would have checked first
		if len(cardList) == 2 {
			for _, c := range cardList {
				possible = append(possible, c)
			}
		}
	}

	if len(possible) >= 4 {
		// Take the largest two pairs
		sort.Sort(sort.Reverse(deck.SortByCards(possible)))
		hand.cards = possible[0:4]
		hand.combo = TwoPair

		// and 1 more card
		for _, c := range cards {
			if !deck.CardInList(c, hand.cards) {
				hand.cards = append(hand.cards, c)
				return hand
			}
		}
	}

	return nil
}

// havePair returns *Hand that makes up a pair, or nil
func havePair(cards []deck.Card) *Hand {

	if len(cards) < 5 {
		log.Fatalf("need at least 5 cards for haveTwoPair, have %d", len(cards))
	}

	var hand = new(Hand)
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	byrank := deck.CountByRank(cards)

	highest := rankCards{}

	// find the highest tuple of cards
	for rank, count := range byrank {
		if count < 2 {
			continue
		}
		if count >= highest.c && rank >= highest.r {
			highest.r = rank
			highest.c = count
		}
	}

	for _, c := range cards {
		if c.GetRank() == highest.r {
			hand.cards = append(hand.cards, c)
		}
	}

	if len(hand.cards) < 2 {
		return nil
	}

	hand.combo = Pair
	// Add 5 more cards
	for _, c := range cards {
		if !deck.CardInList(c, hand.cards) {
			hand.cards = append(hand.cards, c)
		}
		if len(hand.cards) == 5 {
			return hand
		}
	}

	return nil
}

// haveHighCard returns *Hand with the top 5 cards in it
func haveHighCard(cards []deck.Card) *Hand {
	var hand = new(Hand)
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))

	hand.cards = cards[0:5]
	hand.combo = HighCard
	return hand
}

// SortHighCard sorts the cards that have no combos
func SortHighCard(cards []deck.Card) []deck.Card {
	sort.Sort(sort.Reverse(deck.SortByCards(cards)))
	return cards
}

// SortPair sorts the cards that contain only a single pair
func SortPair(cards []deck.Card) []deck.Card {
	var sorted = make([]deck.Card, 2)
	// the pair goes first
	byrank := deck.CardsByRank(cards)

	for _, cardList := range byrank {
		switch len(cardList) {
		case 2:
			for i := 0; i < 2; i++ {
				sorted[i] = cardList[i]
			}
		default:
			for _, c := range cardList {
				sorted = append(sorted, c)
			}
		}
	}

	// sort each section in order
	sort.Sort(sort.Reverse(deck.SortByCards(sorted[2:])))

	return sorted
}

// SortTwoPair sorts the cards that contain two pairs
func SortTwoPair(cards []deck.Card) []deck.Card {
	var sorted = make([]deck.Card, 5)
	// the pairs go first
	byrank := deck.CardsByRank(cards)

	// keep track of which is the next free index in sorted
	nexti := 0

	for _, cardList := range byrank {
		switch len(cardList) {
		case 2:
			for i := 0; i < 2; i++ {
				sorted[nexti] = cardList[i]
				nexti++
			}
		case 1:
			sorted[4] = cardList[0] // only one card left over
		}
	}

	// sort each section in order
	sort.Sort(sort.Reverse(deck.SortByCards(sorted[0:4])))

	return sorted
}

// SortThreeOfAKind sorts the cards that contain three of a kind
func SortThreeOfAKind(cards []deck.Card) []deck.Card {
	var sorted = make([]deck.Card, 3)
	// three of a kind go first
	byrank := deck.CardsByRank(cards)

	for _, cardList := range byrank {
		switch len(cardList) {
		case 3:
			for i := 0; i < 3; i++ {
				sorted[i] = cardList[i] // there are four of these
			}
		default:
			for _, c := range cardList {
				sorted = append(sorted, c)
			}
		}
	}

	// sort each section in order
	sort.Sort(sort.Reverse(deck.SortByCards(sorted[0:3])))
	sort.Sort(sort.Reverse(deck.SortByCards(sorted[3:])))

	return sorted
}

// SortStraight sorts the cards that contain a straight
func SortStraight(cards []deck.Card) []deck.Card {
	sorted := SortHighCard(cards)

	// Handle Ace
	if deck.RankInList(ppb.CardRank_Ace, cards) {
		switch {
		// If there is a two, the Ace should go at the end
		case deck.RankInList(ppb.CardRank_Two, cards):
			ace := sorted[0]
			sorted = sorted[1:]
			sorted = append(sorted, ace)
		}
	}

	return sorted
}

// SortFlush sorts the cards that contain a flush
func SortFlush(cards []deck.Card) []deck.Card {
	return SortHighCard(cards)
}

// SortFullHouse sorts the cards that contain a full house
func SortFullHouse(cards []deck.Card) []deck.Card {
	// full house sorts just like three of a kind, triplet goes first
	return SortThreeOfAKind(cards)
}

// SortFourOfAKind sorts the cards that contain four of a kind
func SortFourOfAKind(cards []deck.Card) []deck.Card {
	var sorted = make([]deck.Card, 5)
	// four of a kind go first
	byrank := deck.CardsByRank(cards)

	for _, cardList := range byrank {
		switch len(cardList) {
		case 4:
			for i := 0; i < 4; i++ {
				sorted[i] = cardList[i] // there are four of these
			}
		case 1:
			sorted[4] = cardList[0] // there is only one
		default:
			log.Fatalf("SortFourOfAKind(%v) did not get 4 of a kind", cards)
		}
	}

	// sort each section in order
	sort.Sort(sort.Reverse(deck.SortByCards(sorted[0:4])))
	return sorted
}

// SortStraightFlush sorts the cards that contain a straight flush
func SortStraightFlush(cards []deck.Card) []deck.Card {
	return SortHighCard(cards)
}

// CompareCards returns -1 if one < two; 0 if one == two; 1 if one > two
// It assumes the hands are sorted and does a card by card comparison
// Once the hands are sorted, this comparison is fine.
func CompareCards(one, two *Hand) int {
	if len(one.cards) != len(two.cards) {
		log.Fatal("pass in same number of cards")
	}

	for i := 0; i < len(one.cards); i++ {
		if one.cards[i].IsSameRank(two.cards[i]) {
			continue
		}

		if one.cards[i].IsLessThan(two.cards[i]) {
			return -1
		}
		return 1
	}

	// all comparisons return same
	return 0
}
