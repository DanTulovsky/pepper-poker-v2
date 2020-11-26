package poker

import (
	"log"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/DanTulovsky/deck"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

var (
	// iterations for random function
	iterations = 450
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	os.Exit(m.Run())
}

func checkCardMatch(t *testing.T, cards []*deck.Card, hc []*deck.Card) {
	sort.Sort(deck.SortByCards(cards))
	sort.Sort(deck.SortByCards(hc))

	for _, c := range hc {
		have := false
		for _, hc := range cards {
			if c.IsSame(hc) {
				have = true
			}
		}
		if !have {
			t.Errorf("cards (%v) and hand.cards (%v) do not match (%v missing)", cards, hc, c)
		}
	}

	var missing int
	for _, c := range cards {
		have := false
		for _, hc := range hc {
			if c.IsSame(hc) {
				have = true
			}
		}
		if !have {
			missing++
		}
	}
	if missing > 2 {
		t.Fatalf("cards (%v) and hand.cards (%v) do not match", cards, hc)
	}
}

func Test_bestCombo(t *testing.T) {
	tests := []struct {
		name   string
		board  []*deck.Card
		player []*deck.Card
		want   *Hand
	}{
		{
			name: "HighCard",
			board: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
			},
			player: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_King),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
					deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_King),
					deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Queen),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				},
				combo: HighCard,
			},
		},
		{
			name: "Straight Ace Low",
			board: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Four),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Two),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Three),
			},
			player: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Jack),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Four),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Two),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Three),
				},
				combo: Straight,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards := append(tt.board, tt.player...)
			got := BestCombo(cards...)
			if got == nil {
				t.Errorf("bestCombo(%v) returned nil", cards)
			}

			if got.combo != tt.want.combo {
				t.Errorf("bestCombo(%v): got %v; expected %v", cards, ComboToString[got.combo], ComboToString[tt.want.combo])
			}
		})
	}
}

func Test_haveFlushRandom(t *testing.T) {
	for i := 0; i < iterations; i++ {
		cards := randomFlush()

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveFlush(cards)
		if hand == nil {
			t.Errorf("haveFlush(%v) expected flush, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveStraightRandom(t *testing.T) {
	for i := 0; i < iterations; i++ {
		cards := randomStraight()

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveStraight(cards)
		if hand == nil {
			t.Fatalf("haveStraight(%v) expected straight, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveStraightFlushRandom(t *testing.T) {
	for i := 0; i < iterations; i++ {
		cards := randomStraightFlush()

		if len(cards) != 5 {
			t.Errorf("expected 5 cards, go: %v", len(cards))
		}

		hand := haveStraightFlush(cards, cards)
		if hand == nil {
			t.Fatalf("haveStraightFlush(%v) expected straight flush, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveFullHouseRandom(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	for i := 0; i < iterations; i++ {
		cards := randomFullHouse()

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveFullHouse(cards)
		if hand == nil {
			t.Fatalf("haveFullHouse(%v) expected full house, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveFourOfAKindRandom(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	for i := 0; i < iterations; i++ {
		cards := randomNOfAKind(4)

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveFourOfAKind(cards)
		if hand == nil {
			t.Fatalf("haveNOfAKind(%v, 4) expected four of a kind, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveThreeOfAKindRandom(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	for i := 0; i < iterations; i++ {
		cards := randomNOfAKind(3)

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveThreeOfAKind(cards)
		if hand == nil {
			t.Fatalf("haveNOfAKind(%v, 3) expected three of a kind, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveTwoPairRandom(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	for i := 0; i < iterations; i++ {
		cards := randomTwoPair()

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveTwoPair(cards)
		if hand == nil {
			t.Fatalf("haveTwoPair(%v) expected two pair, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_havePairRandom(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	for i := 0; i < iterations; i++ {
		cards := randomPair()

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := havePair(cards)
		if hand == nil {
			t.Fatalf("havePair(%v) expected pair, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveHighCardRandom(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	for i := 0; i < iterations; i++ {
		cards := randomHighCard()

		if len(cards) != 7 {
			t.Errorf("expected 7 cards, go: %v", len(cards))
		}

		hand := haveHighCard(cards)
		if hand == nil {
			t.Fatalf("haveHighCard(%v) expected high card, but got nil", cards)
		}

		if len(hand.cards) != 5 {
			t.Fatalf("expected 5 cards, have: %v (%v)", len(hand.cards), hand.cards)
		}

		checkCardMatch(t, cards, hand.cards)
	}
}

func Test_haveStraightFlush(t *testing.T) {
	tests := []struct {
		name    string
		cards   []*deck.Card
		want    *Hand
		wantErr bool
	}{
		{
			name: "HighCard",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_King),
			},
			want: nil,
		},
		{
			name: "Normal Straight Flush",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ten),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ten),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Nine),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Seven),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
				},
				combo: Straight,
			},
		},
		{
			name: "Ace low Straight",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Four),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Two),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Four),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Three),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Two),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				},
				combo: Straight,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := haveStraight(tt.cards)

			if got == nil && tt.want != nil {
				t.Fatalf("haveStraight(%v) returned nil; expected: %v", tt.cards, tt.want)
			}

			if got != nil && tt.want == nil {
				t.Fatalf("haveStraight(%v) did not return nil; expected: %v", tt.cards, tt.want)
			}

			if got != nil && tt.want != nil && got.combo != tt.want.combo {
				t.Errorf("haveStraight(%v) returned combo: %v; expected: %v", tt.cards, ComboToString[got.combo], ComboToString[tt.want.combo])
			}

			if got != nil {
				checkCardMatch(t, tt.cards, got.cards)
			}
		})
	}
}
func Test_haveFlush(t *testing.T) {
	tests := []struct {
		name    string
		cards   []*deck.Card
		want    *Hand
		wantErr bool
	}{
		{
			name: "HighCard",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_King),
			},
			want: nil,
		},
		{
			name: "RoyalFlush",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_King),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Jack),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_King),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_King),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Queen),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Jack),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
				},
				combo: Flush,
			},
		},
		{
			name: "FlushSpades",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Two),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Jack),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_King),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_King),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Jack),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Two),
				},
				combo: Flush,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := haveFlush(tt.cards)

			if got == nil && tt.want != nil {
				t.Fatalf("haveFlush(%v) returned nil; expected: %v", tt.cards, tt.want)
			}

			if got != nil && tt.want == nil {
				t.Fatalf("haveFlush(%v) did not return nil; expected: %v", tt.cards, tt.want)
			}

			if got != nil && tt.want != nil && got.combo != tt.want.combo {
				t.Errorf("haveFlush(%v) returned combo: %v; expected: %v", tt.cards, ComboToString[got.combo], ComboToString[tt.want.combo])
			}

			if got != nil {
				checkCardMatch(t, tt.cards, got.cards)
			}
		})
	}
}

func Test_haveStraight(t *testing.T) {
	tests := []struct {
		name    string
		cards   []*deck.Card
		want    *Hand
		wantErr bool
	}{
		{
			name: "HighCard",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_King),
			},
			want: nil,
		},
		{
			name: "Normal Straight",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
					deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
				},
				combo: Straight,
			},
		},
		{
			name: "Ace low Straight",
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Four),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Two),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
			},
			want: &Hand{
				cards: []*deck.Card{
					deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Four),
					deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Three),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Two),
					deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
				},
				combo: Straight,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := haveStraight(tt.cards)

			if got == nil && tt.want != nil {
				t.Fatalf("haveStraight(%v) returned nil; expected: %v", tt.cards, tt.want)
			}

			if got != nil && tt.want == nil {
				t.Fatalf("haveStraight(%v) did not return nil; expected: %v", tt.cards, tt.want)
			}

			if got != nil && tt.want != nil && got.combo != tt.want.combo {
				t.Errorf("haveStraight(%v) returned combo: %v; expected: %v", tt.cards, ComboToString[got.combo], ComboToString[tt.want.combo])
			}

			if got != nil {
				checkCardMatch(t, tt.cards, got.cards)
			}
		})
	}
}
func TestBestHand(t *testing.T) {
	tests := []struct {
		name        string
		playerHands []*PlayerHand
		want        [][]int // list of levels of index of player that will win
	}{
		// {
		// 	name: "test1",
		// 	playerHands: []*PlayerHand{
		// 		{
		// 			Cards: randomFlush(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomFullHouse(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomHighCard(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomPair(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomTwoPair(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 	},
		// 	want: [][]int{{1}, {0}, {4}, {3}, {2}},
		// },
		{
			name: "test1.1",
			playerHands: []*PlayerHand{
				{
					Cards: randomFlush(),
					ID:    uuid.New().String(),
				},
				{
					Cards: randomFullHouse(),
					ID:    uuid.New().String(),
				},
				{
					Cards: randomStraightFlush(),
					ID:    uuid.New().String(),
				},
				{
					Cards: randomPair(),
					ID:    uuid.New().String(),
				},
				{
					Cards: randomTwoPair(),
					ID:    uuid.New().String(),
				},
			},
			want: [][]int{{2}, {1}, {0}, {4}, {3}},
		},
		// {
		// 	name: "test2",
		// 	playerHands: []*PlayerHand{
		// 		{
		// 			Cards: randomHighCard(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: []*deck.Card{
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Jack),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
		// 				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
		// 			},
		// 			ID: uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: []*deck.Card{
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Jack),
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Eight),
		// 				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Eight),
		// 			},
		// 			ID: uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomPair(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 	},
		// 	want: [][]int{{2, 1}, {3}, {0}},
		// },
		// {
		// 	name: "test3",
		// 	playerHands: []*PlayerHand{
		// 		{
		// 			Cards: randomHighCard(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: []*deck.Card{
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
		// 				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
		// 			},
		// 			ID: uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomPair(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: []*deck.Card{
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_King),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
		// 				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
		// 			},
		// 			ID: uuid.New().String(),
		// 		},
		// 	},
		// 	want: [][]int{{3}, {1}, {2}, {0}},
		// },
		// {
		// 	name: "test4",
		// 	playerHands: []*PlayerHand{
		// 		{
		// 			Cards: randomTwoPair(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: []*deck.Card{
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
		// 				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
		// 				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
		// 			},
		// 			ID: uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: randomPair(),
		// 			ID:    uuid.New().String(),
		// 		},
		// 		{
		// 			Cards: []*deck.Card{
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Four),
		// 				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Five),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Two),
		// 				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
		// 				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Three),
		// 			},
		// 			ID: uuid.New().String(),
		// 		},
		// 	},
		// 	want: [][]int{{1}, {3}, {0}, {2}},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantWinners := []Winners{}
			for _, ilist := range tt.want {
				win := Winners{}
				for _, i := range ilist {
					win = append(win, tt.playerHands[i].ID)
				}
				wantWinners = append(wantWinners, win)
			}
			for _, w := range wantWinners {
				sort.Strings(w)
			}

			got := BestHand(tt.playerHands)
			if !reflect.DeepEqual(got, wantWinners) {
				t.Error("wantWinners")
				for _, w := range wantWinners {
					t.Error(w)
				}
				t.Error("got")
				for _, w := range got {
					t.Error(w)
				}
				t.Error("playerHands")
				for _, h := range tt.playerHands {
					t.Error(h)
				}
				t.Errorf("BestHand() = %v, want %v", got, wantWinners)
			}
		})
	}
}

func TestHand_CompareTo(t *testing.T) {
	type fields struct {
		cards []*deck.Card
		combo Combo
	}
	type args struct {
		other *Hand
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			fields: fields{
				cards: []*deck.Card{
					{},
				},
				combo: HighCard,
			},
			args: args{
				other: &Hand{
					cards: []*deck.Card{
						{},
					},
					combo: HighCard,
				}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Hand{
				cards: tt.fields.cards,
				combo: tt.fields.combo,
			}
			if got := h.CompareTo(tt.args.other); got != tt.want {
				t.Errorf("Hand.CompareTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortThreeOfAKind(t *testing.T) {
	tests := []struct {
		name  string
		cards []*deck.Card
		want  []*deck.Card
	}{
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortThreeOfAKind(tt.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortThreeOfAKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortFourOfAKind(t *testing.T) {
	tests := []struct {
		name  string
		cards []*deck.Card
		want  []*deck.Card
	}{
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortFourOfAKind(tt.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortFourOfAKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortFullHouse(t *testing.T) {
	tests := []struct {
		name  string
		cards []*deck.Card
		want  []*deck.Card
	}{
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
			},
		},
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Six),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortFullHouse(tt.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortFullHouse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortStraight(t *testing.T) {
	tests := []struct {
		name  string
		cards []*deck.Card
		want  []*deck.Card
	}{
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Ten),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Seven),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Six),
			},
		},
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Four),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Two),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Five),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Five),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Four),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Three),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Two),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortStraight(tt.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortFullHouse() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestSortTwoPair(t *testing.T) {
	tests := []struct {
		name  string
		cards []*deck.Card
		want  []*deck.Card
	}{
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortTwoPair(tt.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortTwoPair() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortPair(t *testing.T) {
	tests := []struct {
		name  string
		cards []*deck.Card
		want  []*deck.Card
	}{
		{
			cards: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
			},
			want: []*deck.Card{
				deck.NewCard(ppb.CardSuit_Heart, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Diamond, ppb.CardRank_Ace),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Queen),
				deck.NewCard(ppb.CardSuit_Spade, ppb.CardRank_Nine),
				deck.NewCard(ppb.CardSuit_Club, ppb.CardRank_Eight),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortPair(tt.cards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortPair() = %v, want %v", got, tt.want)
			}
		})
	}
}
