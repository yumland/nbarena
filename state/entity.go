package state

import (
	"github.com/yumland/clone"
	"github.com/yumland/yumbattle/draw"
)

type Entity struct {
	appearance draw.Node

	tilePos       TilePos
	futureTilePos TilePos

	isOwnedByAnswerer bool

	isFlipped bool

	hp        int
	displayHP *int

	canStepOnHoleLikeTiles bool
	ignoresTileEffects     bool
	cannotFlinch           bool
	fatalHitLeaves1HP      bool

	isParalyzed         bool
	paralyzedFramesLeft uint16

	isConfused         bool
	confusedFramesLeft uint16

	isBlinded         bool
	blindedFramesLeft uint16

	isImmobilized         bool
	immobilizedFramesLeft uint16

	isFlashing         bool
	flashingFramesLeft uint16

	isInvincible         bool
	invincibleFramesLeft uint16

	isFrozen         bool
	frozenFramesLeft uint16

	isBubbled         bool
	bubbledFramesLeft uint16

	isBeingDragged bool
}

func (e *Entity) Clone() *Entity {
	return &Entity{
		e.appearance, // Appearances are not cloned: they are considered immutable enough.
		e.tilePos, e.futureTilePos,
		e.isOwnedByAnswerer,
		e.isFlipped,
		e.hp, clone.Shallow(e.displayHP),
		e.canStepOnHoleLikeTiles, e.ignoresTileEffects, e.cannotFlinch, e.fatalHitLeaves1HP,
		e.isParalyzed, e.paralyzedFramesLeft,
		e.isConfused, e.confusedFramesLeft,
		e.isBlinded, e.blindedFramesLeft,
		e.isImmobilized, e.immobilizedFramesLeft,
		e.isFlashing, e.flashingFramesLeft,
		e.isInvincible, e.invincibleFramesLeft,
		e.isFrozen, e.frozenFramesLeft,
		e.isBubbled, e.bubbledFramesLeft,
		e.isBeingDragged,
	}
}

func (e *Entity) TilePos() TilePos {
	return e.tilePos
}

func (e *Entity) SetTilePos(tilePos TilePos) {
	e.tilePos = tilePos
}

func (e *Entity) HP() int {
	return e.hp
}

func (e *Entity) SetHP(hp int) {
	e.hp = hp
}

func (e *Entity) CanStepOnHoleLikeTiles() bool {
	return e.canStepOnHoleLikeTiles
}

func (e *Entity) IgnoresTileEffects() bool {
	return e.ignoresTileEffects
}

func (e *Entity) Appearance() draw.Node {
	return e.appearance
}

func (e *Entity) Step() {
	// TODO: Handle action.

	// Tick timers.
	if !e.isBeingDragged /* && !e.isFrozen */ {
		if e.paralyzedFramesLeft > 0 {
			e.paralyzedFramesLeft--
		}
		if e.confusedFramesLeft > 0 {
			e.confusedFramesLeft--
		}
		if e.blindedFramesLeft > 0 {
			e.blindedFramesLeft--
		}
		if e.immobilizedFramesLeft > 0 {
			e.immobilizedFramesLeft--
		}
		if e.flashingFramesLeft > 0 {
			e.flashingFramesLeft--
		}
		if e.invincibleFramesLeft > 0 {
			e.invincibleFramesLeft--
		}
		if e.frozenFramesLeft > 0 {
			e.frozenFramesLeft--
		}
		if e.bubbledFramesLeft > 0 {
			e.bubbledFramesLeft--
		}
	}

	if e.paralyzedFramesLeft > 0 {
		e.isParalyzed = true
	}
	if e.confusedFramesLeft > 0 {
		e.isConfused = true
	}
	if e.blindedFramesLeft > 0 && !e.isBlinded {
		// Must set flag explicitly.
		e.blindedFramesLeft = 0
	}
	if e.immobilizedFramesLeft > 0 && !e.isImmobilized {
		// Must set flag explicitly.
		e.immobilizedFramesLeft = 0
	}
	if e.flashingFramesLeft > 0 {
		e.isFlashing = true
	}
	if e.invincibleFramesLeft > 0 {
		e.isInvincible = true
	}
	if e.frozenFramesLeft > 0 {
		e.isFrozen = true
	}
	if e.frozenFramesLeft > 0 {
		e.isFrozen = true
	}
}
