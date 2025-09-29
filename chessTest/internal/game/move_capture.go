// path: chessTest/internal/game/move_capture.go
package game

func (ms *MoveState) captureCount() int {
	if ms == nil {
		return 0
	}
	captures := ms.abilityCounter(AbilityNone, abilityCounterCaptures)
	if captures == 0 && len(ms.Captures) > 0 {
		return len(ms.Captures)
	}
	return captures
}

func (ms *MoveState) captureLimit() int {
	if ms == nil {
		return 0
	}
	return ms.abilityCounter(AbilityNone, abilityCounterCaptureLimit)
}

func (ms *MoveState) setCaptureLimit(limit int) {
	if ms == nil {
		return
	}
	if limit < 0 {
		limit = 0
	}
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureLimit, limit)
}

func (ms *MoveState) increaseCaptureLimit(delta int) {
	if ms == nil || delta == 0 {
		return
	}
	current := ms.captureLimit()
	ms.setCaptureLimit(current + delta)
}

func (ms *MoveState) extraRemovalConsumed() bool {
	if ms == nil {
		return false
	}
	return ms.abilityFlag(AbilityNone, abilityFlagCaptureExtra)
}

func (ms *MoveState) resetExtraRemoval() {
	if ms == nil {
		return
	}
	ms.setAbilityFlag(AbilityNone, abilityFlagCaptureExtra, false)
}

func (ms *MoveState) markExtraRemovalConsumed() {
	if ms == nil {
		return
	}
	ms.setAbilityFlag(AbilityNone, abilityFlagCaptureExtra, true)
}

func (ms *MoveState) canCaptureMore() bool {
	if ms == nil {
		return false
	}

	captures := ms.captureCount()
	if captures == 0 {
		return true
	}

	limit := ms.captureLimit()
	if limit <= 0 {
		return true
	}
	return captures < limit
}

func (ms *MoveState) registerCapture(meta SegmentMetadata) {
	if meta.Capture == nil {
		return
	}

	ms.Captures = append(ms.Captures, meta.Capture)

	ms.addAbilityCounter(AbilityNone, abilityCounterCaptures, 1)

	step := len(ms.Path)
	if step >= 2 {
		step -= 2
	} else {
		step = 0
	}
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureSegment, step)
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureSquare, int(meta.CaptureSquare))
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureEnPassant, boolToInt(meta.EnPassant))
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (e *Engine) checkPostCaptureTermination(pc *Piece, target *Piece) bool {
	if e.currentMove == nil {
		return false
	}
	return e.currentMove.TurnEnded
}
