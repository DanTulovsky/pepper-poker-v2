package table

import (
	"fmt"
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

type waitingPlayersState struct {
	baseState

	cache string

	// time to wait before starting the game after the last joined player
	gameWaitTimeout time.Duration

	// the time when the last player was added
	lastPlayerAddedTime time.Time
}

func (i *waitingPlayersState) StartGame() error {
	return fmt.Errorf("game [%v] waiting for players", i.table.ID)
}

func (i *waitingPlayersState) Tick() error {
	i.l.Debugf("Tick(%v)", i.Name())

	now := time.Now()
	var status string

	numActivePlayers := i.table.numActivePlayers()

	status = fmt.Sprintf("Table [%v] waiting for players... (players: %d)", i.table.ID, numActivePlayers)

	if numActivePlayers >= i.table.minPlayers {
		wait := i.gameWaitTimeout - now.Sub(i.lastPlayerAddedTime)
		i.table.gameStartsInTime = wait

		if wait > 0 {
			status = fmt.Sprintf("Table [%v] waiting %v before starting... (players: %d)", i.table.ID, wait.Truncate(time.Second), numActivePlayers)
		} else {
			i.table.setState(i.table.initializingState)
			i.cache = ""
			return nil
		}
	}

	if status != i.cache {
		i.l.Info(status)
		i.cache = status
	}

	time.Sleep(time.Second * 2)

	return nil
}

// AvailableToJoin returns true if the table has empty positions
func (i *waitingPlayersState) AvailableToJoin() bool {
	return i.table.numActivePlayers() < i.table.maxPlayers
}

// AddPlayer adds the player to the table and returns the position at the table
func (i *waitingPlayersState) AddPlayer(player *player.Player) (pos int, err error) {
	i.lastPlayerAddedTime = time.Now()

	return i.table.addPlayer(player)
}

// Reset resets for next roung
func (i *waitingPlayersState) Reset() {
	i.lastPlayerAddedTime = time.Now()
}
