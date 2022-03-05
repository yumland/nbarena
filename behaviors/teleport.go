package behaviors

import (
	"github.com/yumland/nbarena/bundle"
	"github.com/yumland/nbarena/draw"
	"github.com/yumland/nbarena/state"
	"github.com/yumland/pngsheet"
)

const teleportEndlagTicks = 6

type Teleport struct {
}

func (eb *Teleport) Clone() state.EntityBehavior {
	return &Teleport{}
}

func (eb *Teleport) Step(e *state.Entity, s *state.State) {
	if e.BehaviorElapsedTime() == 3 {
		e.FinishMove()
	}

	if e.BehaviorElapsedTime() == 6+teleportEndlagTicks {
		e.SetBehavior(&Idle{})
	}
}

func (eb *Teleport) Appearance(e *state.Entity, b *bundle.Bundle) draw.Node {
	var frame *pngsheet.Frame
	if e.BehaviorElapsedTime() < 3 {
		frame = b.MegamanSprites.TeleportStartAnimation.Frames[e.BehaviorElapsedTime()]
	} else if e.BehaviorElapsedTime() < 6 {
		frame = b.MegamanSprites.TeleportEndAnimation.Frames[e.BehaviorElapsedTime()-3]
	} else {
		frame = b.MegamanSprites.TeleportEndAnimation.Frames[len(b.MegamanSprites.TeleportEndAnimation.Frames)-1]
	}
	return draw.ImageWithFrame(b.MegamanSprites.Image, frame)
}

func (eb *Teleport) Interrupts(e *state.Entity) state.EntityBehaviorInterrupts {
	return state.EntityBehaviorInterrupts{}
}
