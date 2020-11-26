package poker

import (
	"testing"
)

type addition struct {
	player string
	bet    int64
	allin  bool
}
type result struct {
	player string
	amount int64
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		// inputs
		additions []addition
		// expectations
		total int64
		bets  []result
	}{
		{"empty", []addition{}, 0, []result{}},
		{"two players",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, true}, {"b", 100, false}},
			187,
			[]result{{"a", 57}, {"b", 130}}},
		{"three players",
			[]addition{{"a", 5, true}, {"b", 20, false}, {"c", 20, false}, {"b", 53, true}, {"c", 84, true}},
			182,
			[]result{{"a", 5}, {"b", 73}, {"c", 104}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPot()
			for _, addition := range tt.additions {
				p.Add(addition.player, addition.bet, addition.allin)
			}

			if got := p.GetTotal(); got != tt.total {
				t.Errorf("GetTotal() = %v, want %v", got, tt.total)
			}
			for _, bet := range tt.bets {
				if got := p.GetBet(bet.player); got != bet.amount {
					t.Errorf("GetBet() for player %v = %v, want %v", bet.player, got, bet.amount)
				}
			}
		})
	}
}

func TestWinnings(t *testing.T) {
	tests := []struct {
		name string
		// inputs
		additions []addition
		rankings  []Winners
		// expectations
		winnings []result
	}{
		{"empty", []addition{}, []Winners{}, []result{}},
		{"one player, main pot forfeited",
			[]addition{{"a", 3, false}, {"a", 8, false}},
			[]Winners{},
			[]result{{"a", 0}}},
		{"one player, main pot won by a (default)",
			[]addition{{"a", 3, false}, {"a", 8, false}},
			[]Winners{{"a"}},
			[]result{{"a", 11}}},
		{"two players, main pot won by b (default)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, false}, {"b", 100, false}},
			[]Winners{{"b"}},
			[]result{{"a", 0}, {"b", 187}}},
		{"two players, main pot won by a (rank)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, false}, {"b", 100, false}},
			[]Winners{{"a"}, {"b"}},
			[]result{{"a", 187}, {"b", 0}}},
		{"two players, main pot split between a/b (rank)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, false}, {"b", 100, false}},
			[]Winners{{"a", "b"}},
			[]result{{"a", 94}, {"b", 93}}},
		{"two players, main pot and b's subpot won by b (default)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, true}, {"b", 100, false}},
			[]Winners{{"b"}},
			[]result{{"a", 0}, {"b", 187}}},
		{"two players, main pot won by a (default), b's subpot forfeited",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, true}, {"b", 100, false}},
			[]Winners{{"a"}},
			[]result{{"a", 114}, {"b", 0}}},
		{"two players, main pot and b's subpot won by b (rank)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, true}, {"b", 100, false}},
			[]Winners{{"b"}, {"a"}},
			[]result{{"a", 0}, {"b", 187}}},
		{"two players, main pot won by a (rank), b's subpot won by b (default)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, true}, {"b", 100, false}},
			[]Winners{{"a"}, {"b"}},
			[]result{{"a", 114}, {"b", 73}}},
		{"two players, main pot split between a/b (rank), b's subpot won by b (default)",
			[]addition{{"a", 25, false}, {"b", 30, false}, {"a", 32, true}, {"b", 100, false}},
			[]Winners{{"a", "b"}},
			[]result{{"a", 57}, {"b", 130}}},
		{"three players, main pot and a/c's subpot and c's subpot won by c (rank)",
			[]addition{{"a", 10, false}, {"b", 5, true}, {"c", 20, false}, {"a", 53, true}, {"c", 84, true}},
			[]Winners{{"c"}, {"a"}, {"b"}},
			[]result{{"a", 0}, {"b", 0}, {"c", 172}}},
		{"three players, main pot and a/c's subpot won by a (rank), c's subpot won by c (default)",
			[]addition{{"a", 10, false}, {"b", 5, true}, {"c", 20, false}, {"a", 53, true}, {"c", 84, true}},
			[]Winners{{"a"}, {"b"}, {"c"}},
			[]result{{"a", 131}, {"b", 0}, {"c", 41}}},
		{"three players, main pot won by b (rank), a/c's subpot won by a (rank), c's subpot won by c (default)",
			[]addition{{"a", 10, false}, {"b", 5, true}, {"c", 20, false}, {"a", 53, true}, {"c", 84, true}},
			[]Winners{{"b"}, {"a"}, {"c"}},
			[]result{{"a", 116}, {"b", 15}, {"c", 41}}},
		{"three players, main pot split between a/b/c (rank), a/c's subpot split betweeen a/c (rank), c's subpot won by c (default)",
			[]addition{{"a", 10, false}, {"b", 5, true}, {"c", 20, false}, {"a", 53, true}, {"c", 84, true}},
			[]Winners{{"a", "b", "c"}},
			[]result{{"a", 63}, {"b", 5}, {"c", 104}}},
		{"three players, main pot and a/c's subpot split between a/c (rank), a's subpot won by a (default)",
			[]addition{{"c", 20, true}, {"b", 5, true}, {"a", 50, true}},
			[]Winners{{"a", "c"}, {"b"}},
			[]result{{"a", 53}, {"b", 0}, {"c", 22}}},
		{"three players, main pot split between a/b (rank), a/c's subpot and a's subpot won by a (rank and default)",
			[]addition{{"c", 20, true}, {"b", 5, true}, {"a", 50, true}},
			[]Winners{{"a", "b"}, {"c"}},
			[]result{{"a", 68}, {"b", 7}, {"c", 0}}},
		{"three players, main pot split between b/c (rank), a/c's subpot won by c (rank), a's subpot won by a (default)",
			[]addition{{"c", 20, true}, {"b", 5, true}, {"a", 50, true}},
			[]Winners{{"b", "c"}, {"a"}},
			[]result{{"a", 30}, {"b", 8}, {"c", 37}}},
		{"three players, main pot split between a/b/c (rank), a/c's subpot split between a/c (rank), a's subpot won by a (default)",
			[]addition{{"c", 20, true}, {"b", 5, true}, {"a", 50, true}},
			[]Winners{{"a", "b", "c"}},
			[]result{{"a", 50}, {"b", 5}, {"c", 20}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPot()
			for _, addition := range tt.additions {
				p.Add(addition.player, addition.bet, addition.allin)
			}
			p.Finalize(tt.rankings)

			for _, winning := range tt.winnings {
				if got, _ := p.GetWinnings(winning.player); got != winning.amount {
					t.Errorf("GetWinnings() for player %v = %v, want %v", winning.player, got, winning.amount)
				}
			}
		})
	}
}
