package main

import (
	"encoding/csv"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/DanTulovsky/deck"
	"github.com/DanTulovsky/pepper-poker-v2/poker"

	ppb "github.com/DanTulovsky/pepper-poker-v2/proto"
)

//   Attribute Information:
//    1) S1 “Suit of card #1”
//       Ordinal (1-4) representing {Hearts, Spades, Diamonds, Clubs}

//    2) C1 “Rank of card #1”
//       Numerical (1-13) representing (Ace, 2, 3, ... , Queen, King)

//    3) S2 “Suit of card #2”
//       Ordinal (1-4) representing {Hearts, Spades, Diamonds, Clubs}

//    4) C2 “Rank of card #2”
//       Numerical (1-13) representing (Ace, 2, 3, ... , Queen, King)

//    5) S3 “Suit of card #3”
//       Ordinal (1-4) representing {Hearts, Spades, Diamonds, Clubs}

//    6) C3 “Rank of card #3”
//       Numerical (1-13) representing (Ace, 2, 3, ... , Queen, King)

//    7) S4 “Suit of card #4”
//       Ordinal (1-4) representing {Hearts, Spades, Diamonds, Clubs}

//    8) C4 “Rank of card #4”
//       Numerical (1-13) representing (Ace, 2, 3, ... , Queen, King)

//    9) S5 “Suit of card #5”
//       Ordinal (1-4) representing {Hearts, Spades, Diamonds, Clubs}

//    10) C5 “Rank of card 5”
//       Numerical (1-13) representing (Ace, 2, 3, ... , Queen, King)

//    11) CLASS “Poker Hand”
//       Ordinal (0-9)

//       0: Nothing in hand; not a recognized poker hand
//       1: One pair; one pair of equal ranks within five cards
//       2: Two pairs; two pairs of equal ranks within five cards
//       3: Three of a kind; three equal ranks within five cards
//       4: Straight; five cards, sequentially ranked with no gaps
//       5: Flush; five cards with the same suit
//       6: Full house; pair + different rank three of a kind
//       7: Four of a kind; four equal ranks within five cards
//       8: Straight flush; straight + flush
//       9: Royal flush; {Ace, King, Queen, Jack, Ten} + flush

var (
	dataFile = flag.String("data_file", "poker-hand-testing.data", "test data file")
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	data, err := ioutil.ReadFile(*dataFile)
	check(err)

	r := csv.NewReader(strings.NewReader(string(data)))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		check(err)

		checkRecord(record)
	}
}

func checkRecord(record []string) {
	wantHand := getHand(record)
	gotHand := poker.BestCombo(wantHand.Cards()...)

	if wantHand.CompareTo(gotHand) != 0 {
		log.Printf("----------------------------------------------")
		log.Println("mismatch: ")
		log.Printf("  wanted: %v\n", wantHand)
		log.Printf("  got: %v\n", gotHand)
		log.Printf("----------------------------------------------")
	}
}

// 1,1,1,13,2,4,2,3,1,12,0
func getHand(record []string) *poker.Hand {
	cards := []*deck.Card{}
	combo := getCombo(record[10])

	for i := 0; i < 9; i = i + 2 {
		c := convCard(record[i], record[i+1])
		cards = append(cards, c)
	}

	hand := poker.NewHandFrom(cards, combo)
	return hand
}

func getCombo(in string) poker.Combo {

	c, err := strconv.Atoi(in)
	check(err)

	switch c {
	case 0:
		return poker.HighCard
	case 1:
		return poker.Pair
	case 2:
		return poker.TwoPair
	case 3:
		return poker.ThreeOfAKind
	case 4:
		return poker.Straight
	case 5:
		return poker.Flush
	case 6:
		return poker.FullHouse
	case 7:
		return poker.FourOfAKind
	case 8:
		return poker.StraightFlush
	default:
		return poker.StraightFlush
	}
}

func convCard(s, r string) *deck.Card {
	//    1) S1 “Suit of card #1”
	//       Ordinal (1-4) representing {Hearts, Spades, Diamonds, Clubs}
	//    2) C1 “Rank of card #1”
	//       Numerical (1-13) representing (Ace, 2, 3, ... , Queen, King)
	var suit ppb.CardSuit
	var rank ppb.CardRank

	sint, err := strconv.Atoi(s)
	check(err)
	rint, err := strconv.Atoi(r)
	check(err)

	switch sint {
	case 1:
		suit = ppb.CardSuit_Heart
	case 2:
		suit = ppb.CardSuit_Spade
	case 3:
		suit = ppb.CardSuit_Diamond
	case 4:
		suit = ppb.CardSuit_Club
	}

	switch rint {
	case 1:
		rank = ppb.CardRank_Ace
	case 2:
		rank = ppb.CardRank_Two
	case 3:
		rank = ppb.CardRank_Three
	case 4:
		rank = ppb.CardRank_Four
	case 5:
		rank = ppb.CardRank_Five
	case 6:
		rank = ppb.CardRank_Six
	case 7:
		rank = ppb.CardRank_Seven
	case 8:
		rank = ppb.CardRank_Eight
	case 9:
		rank = ppb.CardRank_Nine
	case 10:
		rank = ppb.CardRank_Ten
	case 11:
		rank = ppb.CardRank_Jack
	case 12:
		rank = ppb.CardRank_Queen
	case 13:
		rank = ppb.CardRank_King
	}

	return deck.NewCard(suit, rank)
}
