package behaviors

import (
	"image"

	"github.com/murkland/nbarena/bundle"
	"github.com/murkland/nbarena/draw"
	"github.com/murkland/nbarena/state"
	"github.com/murkland/nbarena/state/query"
)

type WindRack struct {
	Damage int
}

func (eb *WindRack) Flip() {
}

func (eb *WindRack) Clone() state.EntityBehavior {
	return &WindRack{
		eb.Damage,
	}
}

func (eb *WindRack) Traits(e *state.Entity) state.EntityBehaviorTraits {
	return state.EntityBehaviorTraits{
		CanBeCountered: true,
	}
}

func (eb *WindRack) Step(e *state.Entity, s *state.State) {
	if e.BehaviorState.ElapsedTime == 0 {
		s.AddDecoration(&state.Decoration{
			Type:    bundle.DecorationTypeWindSlash,
			TilePos: e.TilePos,
			Offset:  image.Point{0, -16},
		})
	} else if e.BehaviorState.ElapsedTime == 9 {
		x, y := e.TilePos.XY()
		dx := query.DXForward(e.IsFlipped)

		var entities []*state.Entity
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+dx, y))...)
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+dx, y+1))...)
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+dx, y-1))...)

		for _, target := range entities {
			if target.FlashingTimeLeft == 0 {
				var h state.Hit
				h.Traits.Drag = state.DragTypeBig
				h.Traits.SlideDirection = e.Facing()
				h.Traits.Element = state.ElementWind
				h.AddDamage(e.MakeDamageAndConsume(eb.Damage))
				target.Hit.Merge(h)
			}
		}

		for i := 1; i <= 3; i++ {
			s.AddEntity(&state.Entity{
				TilePos: state.TilePosXY(x+dx, i),

				IsFlipped:            e.IsFlipped,
				IsAlliedWithAnswerer: e.IsAlliedWithAnswerer,

				Traits: state.EntityTraits{
					CanStepOnHoleLikeTiles: true,
					IgnoresTileEffects:     true,
					CannotFlinch:           true,
					IgnoresTileOwnership:   true,
				},

				BehaviorState: state.EntityBehaviorState{
					Behavior: &Gust{e.Facing()},
				},
			})
		}
	} else if e.BehaviorState.ElapsedTime == 27-1 {
		e.NextBehavior = &Idle{}
	}
}

func (eb *WindRack) Appearance(e *state.Entity, b *bundle.Bundle) draw.Node {
	rootNode := &draw.OptionsNode{}
	megamanFrameIdx := int(e.BehaviorState.ElapsedTime)
	if megamanFrameIdx >= len(b.MegamanSprites.SlashAnimation.Frames) {
		megamanFrameIdx = len(b.MegamanSprites.SlashAnimation.Frames) - 1
	}
	rootNode.Children = append(rootNode.Children, draw.ImageWithFrame(b.MegamanSprites.Image, b.MegamanSprites.SlashAnimation.Frames[megamanFrameIdx]))

	swordNode := &draw.OptionsNode{Layer: 6}
	rootNode.Children = append(rootNode.Children, swordNode)
	rackFrameIdx := int(e.BehaviorState.ElapsedTime)
	if rackFrameIdx >= len(b.WindRackSprites.Animations[0].Frames) {
		rackFrameIdx = len(b.WindRackSprites.Animations[0].Frames) - 1
	}
	swordNode.Children = append(swordNode.Children, draw.ImageWithFrame(b.WindRackSprites.Image, b.WindRackSprites.Animations[0].Frames[rackFrameIdx]))

	return rootNode
}
