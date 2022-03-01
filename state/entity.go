package state

import (
	"flag"
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yumland/yumbattle/bundle"
	"github.com/yumland/yumbattle/draw"
)

var (
	debugDrawEntityMarker = flag.Bool("debug_draw_entity_markers", false, "draw entity markers")
)

type Damage struct {
	Base int

	ParalyzeTime Ticks
	DoubleDamage bool
	AttackPlus   int
}

type Hit struct {
	TotalDamage int

	FlashTime      Ticks
	ParalyzeTime   Ticks
	ConfuseTime    Ticks
	BlindTime      Ticks
	ImmobilizeTime Ticks
	FreezeTime     Ticks
	BubbleTime     Ticks

	// ???
	Drag bool
}

func (h *Hit) AddDamage(d Damage) {
	v := d.Base + d.AttackPlus
	if d.DoubleDamage {
		v *= 2
	}
	h.TotalDamage += v
	if d.ParalyzeTime > 0 {
		h.ParalyzeTime = d.ParalyzeTime
	}
}

func (h *Hit) Merge(h2 Hit) {
	h.TotalDamage += h2.TotalDamage

	// TODO: Verify this is correct behavior.
	h.ParalyzeTime = h2.ParalyzeTime
	h.ConfuseTime = h2.ConfuseTime
	h.BlindTime = h2.BlindTime
	h.ImmobilizeTime = h2.ImmobilizeTime
	h.FreezeTime = h2.FreezeTime
	h.BubbleTime = h2.BubbleTime
}

type Entity struct {
	id int

	elapsedTime Ticks

	behaviorElapsedTime Ticks
	behavior            EntityBehavior

	tilePos       TilePos
	futureTilePos TilePos

	isAlliedWithAnswerer bool

	isFlipped bool

	isDeleted bool

	hp        int
	displayHP int

	canStepOnHoleLikeTiles bool
	ignoresTileEffects     bool
	cannotFlinch           bool
	fatalHitLeaves1HP      bool
	ignoresTileOwnership   bool

	chargingElapsedTime Ticks
	powerShotChargeTime Ticks

	paralyzedTimeLeft   Ticks
	confusedTimeLeft    Ticks
	blindedTimeLeft     Ticks
	immobilizedTimeLeft Ticks
	flashingTimeLeft    Ticks
	invincibleTimeLeft  Ticks
	frozenTimeLeft      Ticks
	bubbledTimeLeft     Ticks

	currentHit Hit

	isAngry        bool
	isBeingDragged bool
	isSliding      bool
}

func (e *Entity) Clone() *Entity {
	return &Entity{
		e.id,
		e.elapsedTime,
		e.behaviorElapsedTime, e.behavior.Clone(),
		e.tilePos, e.futureTilePos,
		e.isAlliedWithAnswerer,
		e.isFlipped,
		e.isDeleted,
		e.hp, e.displayHP,
		e.canStepOnHoleLikeTiles, e.ignoresTileEffects, e.cannotFlinch, e.fatalHitLeaves1HP, e.ignoresTileOwnership,
		e.chargingElapsedTime, e.powerShotChargeTime,
		e.paralyzedTimeLeft,
		e.confusedTimeLeft,
		e.blindedTimeLeft,
		e.immobilizedTimeLeft,
		e.flashingTimeLeft,
		e.invincibleTimeLeft,
		e.frozenTimeLeft,
		e.bubbledTimeLeft,
		e.currentHit,
		e.isAngry,
		e.isBeingDragged,
		e.isSliding,
	}
}

func (e *Entity) SetBehavior(behavior EntityBehavior) {
	e.behaviorElapsedTime = 0
	e.behavior = behavior
}

func (e *Entity) TilePos() TilePos {
	return e.tilePos
}

func (e *Entity) StartMove(tilePos TilePos, field *Field) bool {
	x, y := tilePos.XY()
	if x < 0 || x >= tileCols || y < 0 || y >= tileRows {
		return false
	}

	tile := &field.tiles[tilePos]
	if tilePos == e.tilePos ||
		(!e.ignoresTileOwnership && e.isAlliedWithAnswerer != tile.isAlliedWithAnswerer) ||
		!tile.CanEnter(e) {
		return false
	}

	e.futureTilePos = tilePos
	return true
}

func (e *Entity) FinishMove() {
	e.tilePos = e.futureTilePos
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

var debugEntityMarkerImage *ebiten.Image
var debugEntityMarkerImageOnce sync.Once

func (e *Entity) Appearance(b *bundle.Bundle) draw.Node {
	rootNode := &draw.OptionsNode{}
	x, y := e.tilePos.XY()
	rootNode.Opts.GeoM.Translate(float64((x-1)*tileRenderedWidth+tileRenderedWidth/2), float64((y-1)*tileRenderedHeight+tileRenderedHeight/2))

	characterNode := &draw.OptionsNode{}
	if e.isFlipped {
		characterNode.Opts.GeoM.Scale(-1, 1)
	}
	if e.frozenTimeLeft > 0 {
		// TODO: Render ice.
		characterNode.Opts.ColorM.Translate(float64(0xa5)/float64(0xff), float64(0xa5)/float64(0xff), float64(0xff)/float64(0xff), 0.0)
	}
	if e.paralyzedTimeLeft > 0 && (e.elapsedTime/2)%2 == 1 {
		characterNode.Opts.ColorM.Translate(1.0, 1.0, 0.0, 0.0)
	}
	if e.flashingTimeLeft > 0 && (e.elapsedTime/2)%2 == 0 {
		characterNode.Opts.ColorM.Translate(0.0, 0.0, 0.0, -1.0)
	}
	characterNode.Children = append(characterNode.Children, e.behavior.Appearance(e, b))

	if e.chargingElapsedTime >= 10 {
		chargingNode := &draw.OptionsNode{}
		characterNode.Children = append(characterNode.Children, chargingNode)

		frames := b.ChargingSprites.ChargingAnimation.Frames
		if e.chargingElapsedTime >= e.powerShotChargeTime {
			frames = b.ChargingSprites.ChargedAnimation.Frames
		}
		frame := frames[int(e.chargingElapsedTime)%len(frames)]
		chargingNode.Children = append(chargingNode.Children, draw.ImageWithFrame(b.ChargingSprites.Image, frame))
	}

	rootNode.Children = append(rootNode.Children, characterNode)

	if *debugDrawEntityMarker {
		debugEntityMarkerImageOnce.Do(func() {
			debugEntityMarkerImage = ebiten.NewImage(5, 5)
			for x := 0; x < 5; x++ {
				debugEntityMarkerImage.Set(x, 2, color.RGBA{0, 255, 0, 255})
			}
			for y := 0; y < 5; y++ {
				debugEntityMarkerImage.Set(2, y, color.RGBA{0, 255, 0, 255})
			}
		})
		rootNode.Children = append(rootNode.Children, draw.ImageWithOrigin(debugEntityMarkerImage, image.Point{2, 2}))
	}

	return rootNode
}

func (e *Entity) Step(sh *StepHandle) {
	e.elapsedTime++

	// Set anger, if required.
	if e.currentHit.TotalDamage >= 300 {
		e.isAngry = true
	}

	// TODO: Process poison damage.

	// Process hit damage.
	mustLeave1HP := e.hp > 1 && e.fatalHitLeaves1HP
	e.hp -= e.currentHit.TotalDamage
	if e.hp < 0 {
		e.hp = 0
	}
	if mustLeave1HP {
		e.hp = 1
	}
	e.currentHit.TotalDamage = 0

	// Tick timers.
	// TODO: Verify this behavior is correct.
	e.behaviorElapsedTime++
	e.behavior.Step(e, sh)

	if !e.currentHit.Drag {
		if !e.isBeingDragged /* && !e.isInTimestop */ {
			// Process flashing.
			if e.currentHit.FlashTime > 0 {
				e.flashingTimeLeft = e.currentHit.FlashTime
				e.currentHit.FlashTime = 0
			}
			if e.flashingTimeLeft > 0 {
				e.flashingTimeLeft--
			}

			// Process paralyzed.
			if e.currentHit.ParalyzeTime > 0 {
				e.paralyzedTimeLeft = e.currentHit.ParalyzeTime
				e.currentHit.ConfuseTime = 0
				e.currentHit.ParalyzeTime = 0
			}
			if e.paralyzedTimeLeft > 0 {
				e.paralyzedTimeLeft--
				e.frozenTimeLeft = 0
				e.bubbledTimeLeft = 0
				e.confusedTimeLeft = 0
			}

			// Process frozen.
			if e.currentHit.FreezeTime > 0 {
				e.frozenTimeLeft = e.currentHit.FreezeTime
				e.paralyzedTimeLeft = 0
				e.currentHit.BubbleTime = 0
				e.currentHit.ConfuseTime = 0
				e.currentHit.FreezeTime = 0
			}
			if e.frozenTimeLeft > 0 {
				e.frozenTimeLeft--
				e.bubbledTimeLeft = 0
				e.confusedTimeLeft = 0
			}

			// Process bubbled.
			if e.currentHit.BubbleTime > 0 {
				e.bubbledTimeLeft = e.currentHit.BubbleTime
				e.confusedTimeLeft = 0
				e.paralyzedTimeLeft = 0
				e.frozenTimeLeft = 0
				e.currentHit.ConfuseTime = 0
				e.currentHit.BubbleTime = 0
			}
			if e.bubbledTimeLeft > 0 {
				e.bubbledTimeLeft--
				e.confusedTimeLeft = 0
			}

			// Process confused.
			if e.currentHit.ConfuseTime > 0 {
				e.confusedTimeLeft = e.currentHit.ConfuseTime
				e.paralyzedTimeLeft = 0
				e.frozenTimeLeft = 0
				e.bubbledTimeLeft = 0
				e.currentHit.FreezeTime = 0
				e.currentHit.BubbleTime = 0
				e.currentHit.ParalyzeTime = 0
				e.currentHit.ConfuseTime = 0
			}
			if e.confusedTimeLeft > 0 {
				e.confusedTimeLeft--
			}

			// Process immobilized.
			if e.currentHit.ImmobilizeTime > 0 {
				e.immobilizedTimeLeft = e.currentHit.ImmobilizeTime
				e.currentHit.ImmobilizeTime = 0
			}
			if e.immobilizedTimeLeft > 0 {
				e.immobilizedTimeLeft--
			}

			// Process blinded.
			if e.currentHit.BlindTime > 0 {
				e.blindedTimeLeft = e.currentHit.BlindTime
				e.currentHit.BlindTime = 0
			}
			if e.blindedTimeLeft > 0 {
				e.blindedTimeLeft--
			}

			// Process invincible.
			if e.invincibleTimeLeft > 0 {
				e.invincibleTimeLeft--
			}
		} else {
			// TODO: Interrupt player.
		}
	} else {
		e.currentHit.Drag = false

		e.frozenTimeLeft = 0
		e.bubbledTimeLeft = 0
		e.paralyzedTimeLeft = 0
		e.currentHit.BubbleTime = 0
		e.currentHit.FreezeTime = 0

		if false {
			e.paralyzedTimeLeft = 0
		}

		// TODO: Interrupt player.
	}

	// Update UI.
	if e.displayHP != 0 && e.displayHP != e.hp {
		if e.hp < e.displayHP {
			e.displayHP -= ((e.displayHP-e.hp)>>3 + 4)
			if e.displayHP < e.hp {
				e.displayHP = e.hp
			}
		} else {
			e.displayHP += ((e.hp-e.displayHP)>>3 + 4)
			if e.displayHP > e.hp {
				e.displayHP = e.hp
			}
		}
	}
}
