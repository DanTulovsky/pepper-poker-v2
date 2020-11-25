package acks

import (
	"time"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
	"github.com/google/uuid"
)

// Token is a single ack token
type Token struct {
	id        string
	acked     map[*player.Player]bool
	mustack   []*player.Player // who must ack
	mustackin time.Duration    // how long they have to ack
	start     time.Time
}

// New creates a new ack token
func New(mustack []*player.Player, mustackin time.Duration) *Token {
	return &Token{
		id:        uuid.New().String(),
		acked:     make(map[*player.Player]bool),
		mustack:   mustack,
		mustackin: mustackin,
		start:     time.Now(),
	}
}

// String returns ...
func (t *Token) String() string {
	return t.id
}

// StartTime starts the ack timer
func (t *Token) StartTime() {
	t.start = time.Now()
}

// Ack records a player acking a token
func (t *Token) Ack(p *player.Player) error {
	t.acked[p] = true
	return nil
}

// TimeRemaining returns time left until token expires
func (t *Token) TimeRemaining() time.Duration {
	return -(time.Now().Sub(t.start) - t.mustackin)
}

// NumStillToAck returns the number of players that still need to ack the token
func (t *Token) NumStillToAck() int {
	return len(t.mustack) - len(t.acked)
}

// HaveAck returns trus if a player acked a token
func (t *Token) HaveAck(p *player.Player) bool {
	if _, ok := t.acked[p]; ok {
		return true
	}
	return false
}

// AllAcked returns true when the token is acked by all players
func (t *Token) AllAcked() bool {

	for _, p := range t.mustack {
		if _, ok := t.acked[p]; !ok {
			return false
		}
	}

	return true
}

// Expired returns true when the token is expired
func (t *Token) Expired() bool {
	if t.TimeRemaining() > 0 {
		return false
	}
	return true
}
