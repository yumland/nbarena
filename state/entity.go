package state

import (
	"flag"
	"image"
	"image/color"
	"strconv"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/clone"
	"github.com/murkland/nbarena/bundle"
	"github.com/murkland/nbarena/draw"
	"golang.org/x/exp/slices"
)

var (
	debugDrawEntityMarker = flag.Bool("debug_draw_entity_markers", false, "draw entity markers")
)

type EntityTraits struct {
	CanStepOnHoleLikeTiles bool
	IgnoresTileEffects     bool
	CannotFlinch           bool
	FatalHitLeaves1HP      bool
	IgnoresTileOwnership   bool
}

type EntityPerTickState struct {
	WasHit                  bool
	DoubleDamageWasConsumed bool
}

type SlideState struct {
	Direction   Direction
	ElapsedTime Ticks
}

type EntityBehaviorTraits struct {
	CanBeCountered bool
	RunsInTimestop bool
}

type EntityBehaviorState struct {
	Behavior    EntityBehavior
	ElapsedTime Ticks
}

func (s EntityBehaviorState) Clone() EntityBehaviorState {
	return EntityBehaviorState{s.Behavior.Clone(), s.ElapsedTime}
}

type EntityID int

type ChipPlaque struct {
	ElapsedTime Ticks
	Chip        *Chip
}

type Entity struct {
	id EntityID

	elapsedTime Ticks

	BehaviorState        EntityBehaviorState
	NextBehavior         EntityBehavior
	IsPendingDestruction bool

	Intent     Intent
	LastIntent Intent

	TilePos       TilePos
	FutureTilePos TilePos

	SlideState SlideState

	IsAlliedWithAnswerer bool

	IsFlipped bool

	IsDead bool

	HP        int
	MaxHP     int
	DisplayHP int

	Traits EntityTraits

	PowerShotChargeTime Ticks

	ConfusedTimeLeft    Ticks
	BlindedTimeLeft     Ticks
	ImmobilizedTimeLeft Ticks
	FlashingTimeLeft    Ticks
	InvincibleTimeLeft  Ticks

	IsAngry       bool
	IsFullSynchro bool

	Hit          Hit
	PerTickState EntityPerTickState

	Chips                  []*Chip
	ChipUseQueued          bool
	ChipUseLockoutTimeLeft Ticks

	ChipPlaque ChipPlaque
}

func (e *Entity) ID() EntityID {
	return e.id
}

func (e *Entity) Flip() {
	e.IsAlliedWithAnswerer = !e.IsAlliedWithAnswerer
	e.IsFlipped = !e.IsFlipped
	e.TilePos = e.TilePos.Flipped()
	e.FutureTilePos = e.FutureTilePos.Flipped()
	e.SlideState.Direction = e.SlideState.Direction.FlipH()
	e.BehaviorState.Behavior.Flip()
}

func (e *Entity) Clone() *Entity {
	return &Entity{
		e.id,
		e.elapsedTime,
		e.BehaviorState.Clone(), clone.Interface[EntityBehavior](e.NextBehavior), e.IsPendingDestruction,
		e.Intent, e.LastIntent,
		e.TilePos, e.FutureTilePos,
		e.SlideState,
		e.IsAlliedWithAnswerer,
		e.IsFlipped,
		e.IsDead,
		e.HP, e.MaxHP, e.DisplayHP,
		e.Traits,
		e.PowerShotChargeTime,
		e.ConfusedTimeLeft, e.BlindedTimeLeft, e.ImmobilizedTimeLeft, e.FlashingTimeLeft, e.InvincibleTimeLeft,
		e.IsAngry, e.IsFullSynchro,
		e.Hit, e.PerTickState,
		slices.Clone(e.Chips), e.ChipUseQueued, e.ChipUseLockoutTimeLeft,
		e.ChipPlaque,
	}
}

func (e *Entity) Facing() Direction {
	if e.IsFlipped {
		return DirectionLeft
	}
	return DirectionRight
}

func (e *Entity) UseChip(s *State) bool {
	if len(e.Chips) == 0 {
		return false
	}
	chip := e.Chips[len(e.Chips)-1]
	e.Chips = e.Chips[:len(e.Chips)-1]
	e.NextBehavior = chip.MakeBehavior()
	e.ChipPlaque = ChipPlaque{Chip: chip}
	return true
}

func (e *Entity) ElapsedTime() Ticks {
	return e.elapsedTime
}

func (e *Entity) MoveDirectly(tilePos TilePos) bool {
	if tilePos < 0 {
		return false
	}

	x, y := tilePos.XY()
	if x < 0 || x >= TileCols || y < 0 || y >= TileRows {
		return false
	}

	e.TilePos = tilePos
	return true
}

func (e *Entity) StartMove(tilePos TilePos, s *State) bool {
	if tilePos < 0 {
		return false
	}

	x, y := tilePos.XY()
	if x < 0 || x >= TileCols || y < 0 || y >= TileRows {
		return false
	}

	tile := s.Field.Tiles[tilePos]
	if tilePos == e.TilePos ||
		(!e.Traits.IgnoresTileOwnership && e.IsAlliedWithAnswerer != tile.IsAlliedWithAnswerer) ||
		(tile.Reserver != 0 && tile.Reserver != e.id) ||
		!tile.CanEnter(e) {
		return false
	}

	// TODO: Figure out when to trigger onleave/onenter callbacks
	tile.Reserver = e.ID()
	e.FutureTilePos = tilePos
	return true
}

func (e *Entity) FinishMove(s *State) {
	// TODO: Trigger on leave?
	s.Field.Tiles[e.TilePos].Reserver = 0
	e.TilePos = e.FutureTilePos
}

// SetBehaviorImmediate sets the entity's behavior immediately to the next state and steps once. You probably don't want to call this: you should probably use NextBehavior instead.
func (e *Entity) SetBehaviorImmediate(behavior EntityBehavior, s *State) {
	e.BehaviorState = EntityBehaviorState{Behavior: behavior}
	e.BehaviorState.Behavior.Step(e, s)
}

var debugEntityMarkerImage *ebiten.Image
var debugEntityMarkerImageOnce sync.Once

func BehaviorIs[T EntityBehavior](behavior EntityBehavior) bool {
	_, ok := behavior.(T)
	return ok
}

func (e *Entity) Appearance(b *bundle.Bundle) draw.Node {
	rootNode := &draw.OptionsNode{}
	x, y := e.TilePos.XY()

	dx, dy := e.SlideState.Direction.XY()
	offset := (int(e.SlideState.ElapsedTime)+2+4)%4 - 2
	dx *= offset
	dy *= offset

	rootNode.Opts.GeoM.Translate(
		float64((x-1)*TileRenderedWidth+TileRenderedWidth/2+dx*TileRenderedWidth/4),
		float64((y-1)*TileRenderedHeight+TileRenderedHeight/2+dy*TileRenderedHeight/4),
	)

	rootCharacterNode := &draw.OptionsNode{}
	rootNode.Children = append(rootNode.Children, rootCharacterNode)

	if e.IsFlipped {
		rootCharacterNode.Opts.GeoM.Scale(-1, 1)
	}

	characterNode := &draw.OptionsNode{}
	rootCharacterNode.Children = append(rootCharacterNode.Children, characterNode)

	characterNode.Children = append(characterNode.Children, e.BehaviorState.Behavior.Appearance(e, b))

	if e.FlashingTimeLeft > 0 && (e.elapsedTime/2)%2 == 0 {
		characterNode.Opts.ColorM.Translate(0.0, 0.0, 0.0, -1.0)
	}
	if e.PerTickState.WasHit {
		characterNode.Opts.ColorM.Translate(1.0, 1.0, 1.0, 0.0)
	}
	if e.IsFullSynchro {
		characterNode.Opts.ColorM.Translate(float64(0x29)/float64(0xff), float64(0x29)/float64(0xff), float64(0x29)/float64(0xff), 0.0)

		fullSynchroNode := &draw.OptionsNode{Layer: 8}
		fullSynchroNode.Children = append(fullSynchroNode.Children, draw.ImageWithAnimation(b.FullSynchroSprites.Image, b.FullSynchroSprites.Animations[0], int(e.elapsedTime)))
		rootCharacterNode.Children = append(rootCharacterNode.Children, fullSynchroNode)
	}

	if *debugDrawEntityMarker {
		debugEntityMarkerImageOnce.Do(func() {
			debugEntityMarkerImage = ebiten.NewImage(5, 5)
			for x := 0; x < 5; x++ {
				debugEntityMarkerImage.Set(x, 2, color.RGBA{255, 255, 255, 255})
			}
			for y := 0; y < 5; y++ {
				debugEntityMarkerImage.Set(2, y, color.RGBA{255, 255, 255, 255})
			}
		})
		debugEntityMarkerNode := &draw.OptionsNode{}
		debugEntityMarkerNode.Children = append(debugEntityMarkerNode.Children, draw.ImageWithOrigin(debugEntityMarkerImage, image.Point{2, 2}))
		debugEntityMarkerNode.Opts.ColorM.Scale(1.0, 0.0, 1.0, 0.5)
		rootNode.Children = append(rootNode.Children, debugEntityMarkerNode)
	}

	if e.IsAlliedWithAnswerer {
		if e.DisplayHP != 0 {
			hpNode := &draw.OptionsNode{}
			rootNode.Children = append(rootNode.Children, hpNode)

			// Render HP.
			hpText := strconv.Itoa(int(e.DisplayHP))
			hpNode.Opts.GeoM.Translate(float64(0), float64(4))

			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					strokeNode := &draw.OptionsNode{}
					hpNode.Children = append(hpNode.Children, strokeNode)
					strokeNode.Opts.GeoM.Translate(float64(dx), float64(dy))
					strokeNode.Opts.ColorM.Scale(float64(0x31)/float64(0xFF), float64(0x39)/float64(0xFF), float64(0x52)/float64(0xFF), 1.0)
					strokeNode.Children = append(strokeNode.Children, &draw.TextNode{Text: hpText, Face: b.TinyNumFont, Anchor: draw.TextAnchorCenter})
				}

				fillNode := &draw.OptionsNode{}
				hpNode.Children = append(hpNode.Children, fillNode)
				if e.DisplayHP > e.HP {
					fillNode.Opts.ColorM.Scale(float64(0xFF)/float64(0xFF), float64(0x84)/float64(0xFF), float64(0x5A)/float64(0xFF), 1.0)
				} else if e.DisplayHP < e.HP {
					fillNode.Opts.ColorM.Scale(float64(0x73)/float64(0xFF), float64(0xFF)/float64(0xFF), float64(0x4A)/float64(0xFF), 1.0)
				}
				fillNode.Children = append(fillNode.Children, &draw.TextNode{Text: hpText, Face: b.TinyNumFont, Anchor: draw.TextAnchorCenter})
			}
		}
	} else {
		chipsNode := &draw.OptionsNode{}
		chipsNode.Opts.GeoM.Translate(0, float64(-56))
		rootNode.Children = append(rootNode.Children, chipsNode)

		for i, chip := range e.Chips {
			chipNode := &draw.OptionsNode{Layer: 8}
			j := len(e.Chips) - i - 1
			chipNode.Opts.GeoM.Translate(float64(-j*2), float64(-j*2))
			chipsNode.Children = append(chipsNode.Children, chipNode)

			chipNode.Children = append(chipNode.Children, draw.ImageWithFrame(b.ChipIconSprites.Image, b.ChipIconSprites.Animations[chip.Index].Frames[0]))
		}
	}

	return rootNode
}

func (e *Entity) Step(s *State) {
	if e.ChipUseLockoutTimeLeft > 0 {
		e.ChipUseLockoutTimeLeft--
	}

	if e.ChipPlaque.Chip != nil {
		e.ChipPlaque.ElapsedTime++
		if e.ChipPlaque.ElapsedTime >= 60 {
			e.ChipPlaque = ChipPlaque{}
		}
	}

	e.elapsedTime++
	// Tick timers.
	// TODO: Verify this behavior is correct.
	e.BehaviorState.ElapsedTime++
	if e.NextBehavior != nil {
		e.BehaviorState = EntityBehaviorState{e.NextBehavior, 0}
	}
	e.NextBehavior = nil
	e.BehaviorState.Behavior.Step(e, s)
}

func (e *Entity) MakeDamageAndConsume(base int) Damage {
	dmg := Damage{
		Base: base,

		DoubleDamage: e.IsAngry || e.IsFullSynchro,
	}
	e.IsAngry = false
	e.IsFullSynchro = false
	if dmg.DoubleDamage {
		e.PerTickState.DoubleDamageWasConsumed = true
	}
	return dmg
}

type EntityBehavior interface {
	clone.Cloner[EntityBehavior]
	Flip()
	Appearance(e *Entity, b *bundle.Bundle) draw.Node
	Traits(e *Entity) EntityBehaviorTraits
	Step(e *Entity, s *State)
}
