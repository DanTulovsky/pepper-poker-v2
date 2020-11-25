package id

import "github.com/google/uuid"

// TableID is a unique id for table
type TableID string

const (
	// EmptyPlayerID is ...
	EmptyPlayerID PlayerID = ""

	// EmptyTableID is ...
	EmptyTableID TableID = ""
)

// NewTableID creates a new id for the table
func NewTableID() TableID {
	return TableID(uuid.New().String())
}

func (id TableID) String() string {
	return string(id)
}

// PlayerID is a unique id for a player
type PlayerID string

// NewPlayerID creates a new id for a player
func NewPlayerID() PlayerID {
	return PlayerID(uuid.New().String())
}

func (id PlayerID) String() string {
	return string(id)
}

// RoundID is a unique id for a round
type RoundID string

// NewRoundID creates a new id for the round
func NewRoundID() RoundID {
	return RoundID(uuid.New().String())
}

func (id RoundID) String() string {
	return string(id)
}
