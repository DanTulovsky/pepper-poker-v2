package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/dustin/go-humanize"
)

type waitingPlayersState struct {
	baseState

	cache string

	// time to wait before starting the game after the last joined player
	gameWaitTimeout time.Duration

	// the time when the last player was added
	lastPlayerAddedTime time.Time
}

func (i *waitingPlayersState) Init() error {
	// add any pending players that are new
	for _, p := range i.table.pendingPlayers {
		if _, err := i.AddPlayer(p); err != nil {
			i.l.Error(err)
		}
	}
	i.table.pendingPlayers = nil

	return fmt.Errorf("game [%v] waiting for players", i.table.ID)
}

func (i *waitingPlayersState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	now := time.Now()
	var status string

	numAvailablePlayers := i.table.numAvailablePlayers()

	status = fmt.Sprintf("Table [%v] waiting for players... (players: %d)", i.table.ID, numAvailablePlayers)

	if numAvailablePlayers >= i.table.minPlayers && i.table.playersReady() {
		wait := i.gameWaitTimeout - now.Sub(i.lastPlayerAddedTime)
		i.table.gameStartsInTime = wait

		if wait > 0 {
			status = fmt.Sprintf("Table [%v] waiting %v before starting... (players: %d; [%v])", i.table.ID, wait.Truncate(time.Second), numAvailablePlayers, i.table.AvailablePlayers())
		} else {
			i.l.Info("Adding players to current hand...")
			for _, p := range i.table.AvailablePlayers() {
				i.table.AddCurrentHandPlayer(p)
			}

			i.table.setState(i.table.initializingState)
			i.cache = ""
			return nil
		}
	}

	if status != i.cache {
		i.l.Info(status)
		i.cache = status
	}

	return nil
}

// AddPlayer adds the player to the table and returns the position at the table
func (i *waitingPlayersState) AddPlayer(p *player.Player) (pos int, err error) {
	i.lastPlayerAddedTime = time.Now()

	if i.table.numPresentPlayers() == i.table.maxPlayers {
		return -1, fmt.Errorf("no available positions at table")
	}

	if !i.table.playerAtTable(p) {
		// buy in
		i.l.Infof("[%v] buying into the table ($%v)", p.Name, humanize.Comma(i.table.buyinAmount))
		if err := i.BuyIn(p); err != nil {
			return -1, err
		}

		i.l.Infof("Addting player [%v] to table [%v]", p.Name, i.table.Name)
		i.table.positions[i.table.nextAvailablePosition()] = p
		p.TablePosition, err = i.table.PlayerPosition(p)
		return i.table.PlayerPosition(p)
	}

	return -1, fmt.Errorf("player already at the table: %v (%v)", p.Name, p.ID)
}

// Reset resets for next roung
func (i *waitingPlayersState) Reset() {
	i.lastPlayerAddedTime = time.Now()
}

func (i *waitingPlayersState) WaitingTurnPlayer() *player.Player {
	return nil
}

// BuyIn process the buyin request
func (i *waitingPlayersState) BuyIn(p *player.Player) error {
	return i.table.buyin(p)
}
