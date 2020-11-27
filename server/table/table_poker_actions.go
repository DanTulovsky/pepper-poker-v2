package table

import (
	"github.com/DanTulovsky/pepper-poker-v2/server/player"
)

// TODO: delete all these
func (t *Table) bet(p *player.Player, bet int64) error {
	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) call(p *player.Player) error {
	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) check(p *player.Player) error {
	// TODO: Additional checks when money is available

	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}

func (t *Table) fold(p *player.Player) error {
	// Success
	p.SetActionRequired(false)
	p.CurrentTurn++
	return nil
}
