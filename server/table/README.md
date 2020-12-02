## Joining Tables

### Table in WaitingPlayers State

Joining puts you directly into an empty position on the table, or returns an error that there are none.

### Any Other State

Joining puts you into the pendingPlayers queue.  During the next waitingPlayersState
you are added to an available position.

## Buying In (new players joining)

### Table in WaitingPlayers State

Buyin happens on join.

### Any Other State

Buyin happens at the next waitingPlayerState

## Buying In (existing players)

Players that lost all of their money remain in ActivePlayers(), but do not participate in the game.

They can send a BuyIn RPC at any point, which increases their stack. They get automatically added
back in during the next waitingPlayerState.
