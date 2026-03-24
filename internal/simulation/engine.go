// Package simulation implements the pure, Firestore-free simulation tick.
// The engine receives a snapshot of world state and returns what changed.
// Zero network calls inside — all Firestore I/O is handled by the caller.
package simulation

import (
	"time"

	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/world"
)

const (
	minDeltaSec  = 1.0  // skip ticks shorter than this
	OutputBufCap = 50   // max items per output slot
	InputBufCap  = 50   // max items per input slot
)

var tierMultipliers = map[int]float64{
	1: 1.0,
	2: 2.0,
	3: 4.0,
}

// WorldState is the read-only snapshot the engine operates on.
// Buildings is a map of deep-copied building documents — the engine mutates them freely.
type WorldState struct {
	World     world.World
	Buildings map[string]*world.Building // buildingID → mutable copy
	Inventory world.Inventory            // mutable copy
	Registry  *recipe.Registry
	BeltTier  int // 1, 2, or 3; 0 treated as 1
}

// SimulationResult contains everything that changed during the tick.
// The caller is responsible for persisting these back to Firestore.
type SimulationResult struct {
	// MutatedBuildings contains all buildings whose state changed.
	MutatedBuildings map[string]*world.Building
	// UpdatedInventory is the full inventory after the tick (not a delta).
	UpdatedInventory world.Inventory
	// NewLastSimulated is the timestamp to store on the world document.
	NewLastSimulated time.Time
	// Skipped is true when the delta was too small to warrant a write.
	Skipped bool
}

// Tick runs one simulation step and returns the result.
// It is a pure function: given the same WorldState and now, it always returns
// the same result. No Firestore calls, no global state.
func Tick(state WorldState, now time.Time, maxOfflineSec int64) SimulationResult {
	delta := now.Sub(state.World.LastSimulatedAt).Seconds()
	if delta < minDeltaSec {
		return SimulationResult{Skipped: true}
	}
	if delta > float64(maxOfflineSec) {
		delta = float64(maxOfflineSec)
	}

	beltSpeedPerMin := beltSpeed(state.BeltTier)

	stepExtractors(state.Buildings, state.Registry, delta)
	stepConveyors(state.Buildings, beltSpeedPerMin, delta)
	stepFactories(state.Buildings, state.Registry, delta)
	stepResearchLabs(state.Buildings, state.Registry, &state.Inventory)

	return SimulationResult{
		MutatedBuildings: state.Buildings,
		UpdatedInventory: state.Inventory,
		NewLastSimulated: now,
	}
}

func beltSpeed(tier int) float64 {
	switch tier {
	case 2:
		return 120
	case 3:
		return 240
	default:
		return 60
	}
}

func ifloor(f float64) int64 {
	if f < 0 {
		return 0
	}
	return int64(f)
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
