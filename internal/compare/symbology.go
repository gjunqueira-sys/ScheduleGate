package compare

import (
	"fmt"
	"math"
)

// Symbols for different change types
const (
	SymNew         = "⊕"
	SymDeleted     = "×"
	SymDelayed     = "☒"
	SymEarly       = "←"
	SymBloated     = "🐢"
	SymGhost       = "👻"
	SymNeutral     = "📝"
	SymFinishLater = "and"
)

// AssignSymbology calculates the symbol, impact type, and message for a task delta.
func AssignSymbology(d *TaskDelta) {
	// 1. Ghost Task always takes priority over status-based symbols.
	if d.IsGhostTask {
		d.Symbol = SymGhost
		d.ImpactType = "Reliability"
		d.ImpactMsg = "Ghost Task (Sliding)"
		return
	}

	// 2. New or Deleted (Scope Churn)
	if d.Status == StatusNew {
		d.Symbol = SymNew
		d.ImpactType = "Scope"
		d.ImpactMsg = "New Task Added"
		return
	}
	if d.Status == StatusDeleted {
		d.Symbol = SymDeleted
		d.ImpactType = "Scope"
		d.ImpactMsg = "Task Deleted"
		return
	}

	// 3. Modified Tasks
	if d.Status == StatusModified {
		if d.FinishVariance > 2.0 {
			d.Symbol = SymDelayed
			d.ImpactType = "Stability"
			d.ImpactMsg = fmt.Sprintf("Delayed %.1fd", d.FinishVariance)
			return
		}

		if d.DurationDelta > 0 && d.PrevDuration > 0 {
			growth := d.DurationDelta / d.PrevDuration
			if growth > 0.10 {
				d.Symbol = SymBloated
				d.ImpactType = "Reliability"
				d.ImpactMsg = fmt.Sprintf("Duration +%.0f%%", growth*100)
				return
			}
		}

		if d.FinishVariance < -1.0 {
			d.Symbol = SymEarly
			d.ImpactType = "Stability"
			d.ImpactMsg = fmt.Sprintf("Pulled in %.1fd", math.Abs(d.FinishVariance))
			return
		}

		d.Symbol = SymNeutral
		d.ImpactType = "Neutral"
		d.ImpactMsg = "Minor modification"
	} else if d.Status == StatusUnchanged {
		d.Symbol = "□"
		d.ImpactType = "Neutral"
		d.ImpactMsg = "No Change"
	}
}
