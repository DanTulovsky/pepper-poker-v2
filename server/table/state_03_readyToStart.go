package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	numPlayers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pepperpoker_players_total",
		Help: "The total number of players",
	}, []string{"type"})
)

type readyToStartState struct {
	baseState
	playerTimeout time.Duration
}

func (i *readyToStartState) Init() error {
	i.baseState.Init()

	i.l.Info("Starting new game with players...")
	numPlayers.WithLabelValues("active").Set(float64(i.table.numActivePlayers()))
	numPlayers.WithLabelValues("available").Set(float64(i.table.numAvailablePlayers()))

	i.l.Info("Dealings cards to players...")
	for j := 0; j < 2; j++ {
		for _, p := range i.table.ActivePlayers() {
			card, err := i.table.deck.Next()
			if err != nil {
				return err
			}

			p.AddHoleCard(card)
		}
	}

	for _, p := range i.table.ActivePlayers() {
		i.l.Infof("  [%v ($%v)]: %v", p.Name, humanize.Comma(p.Money().Stack()), p.Hole())
	}

	return nil
}

func (i *readyToStartState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	i.table.setState(i.table.playingSmallBlindState)
	return nil
}

func (i *readyToStartState) AddPlayer(player *player.Player) (pos int, err error) {
	return -1, fmt.Errorf("game already started, wait for next round")
}

// WhoseTurn returns the player whose turn it is.
func (i *readyToStartState) WhoseTurn() *player.Player {
	return nil
}

func (i *readyToStartState) WaitingTurnPlayer() *player.Player {
	return nil
}
