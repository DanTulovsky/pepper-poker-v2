package poker

import (
	"fmt"
	"strings"

	"github.com/DanTulovsky/deck"
)

// Board holds the community cards
type Board struct {
	cards []*deck.Card
}

// NewBoard returns a new board with no cards
func NewBoard() *Board {
	return &Board{
		cards: []*deck.Card{},
	}
}

// Cards returns the cards from the board
func (b *Board) Cards() []*deck.Card {
	return b.cards
}

func (b *Board) String() string {
	var output strings.Builder

	for _, c := range b.cards {
		output.WriteString(fmt.Sprintf("%v", c))
	}

	return output.String()
}
