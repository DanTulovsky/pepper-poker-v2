package table

import (
	"fmt"

	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

func (t *Table) bet(p *player.Player, bet int64) error {

	if t.State.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")

	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}

	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	return nil
}

func (t *Table) call(p *player.Player) error {

	if t.State.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")

	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}

	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	return nil
}

func (t *Table) check(p *player.Player) error {
	if t.State.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")

	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}

	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	return nil
}

func (t *Table) fold(p *player.Player) error {

	if t.State.WaitingTurnPlayer() != p {
		return fmt.Errorf("it's not your turn")

	}
	if !p.ActionRequired() {
		return fmt.Errorf("no action required from you")
	}

	// Success
	p.SetActionRequired(false)
	return nil
}
