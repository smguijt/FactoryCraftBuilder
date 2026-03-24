package simulation

import (
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/world"
)

// stepConveyors moves items from each building's output buffer to its downstream
// real buildings (skipping conveyor tiles). Buildings are processed in topological
// order so items flow correctly through chains of producers.
func stepConveyors(buildings map[string]*world.Building, beltSpeedPerMin, delta float64) {
	throughputCap := ifloor(beltSpeedPerMin / 60.0 * delta)
	if throughputCap == 0 {
		return
	}

	order := topoSort(buildings)

	for _, id := range order {
		src := buildings[id]
		if len(src.Connections) == 0 {
			continue
		}
		// Conveyors are purely routing — no buffer, nothing to drain.
		if src.Type == recipe.BuildingConveyor {
			continue
		}
		// isActive only gates production, not output drainage.
		// Items already in an output buffer always drain onto connected belts.

		if src.Type == recipe.BuildingSplitter {
			stepSplitter(src, buildings, throughputCap)
		} else {
			// Single-output: follow belt chain to first real destination
			dest := realDest(src.Connections[0], buildings)
			if dest == nil {
				continue
			}
			transferItems(src, dest, throughputCap)
		}
	}
}

// transferItems moves up to cap items of each type from src.OutputSlots → dst.InputSlots.
func transferItems(src, dst *world.Building, cap int64) {
	// Research lab has no input cap — items are consumed instantly
	isResearchLab := dst.Type == recipe.BuildingResearchLab

	for itemID, qty := range src.OutputSlots {
		if qty == 0 {
			continue
		}
		var space int64
		if isResearchLab {
			space = InputBufCap // always accepts up to cap
		} else {
			space = InputBufCap - dst.InputSlots[itemID]
		}
		if space <= 0 {
			continue
		}
		move := min64(qty, min64(cap, space))
		if move <= 0 {
			continue
		}
		src.OutputSlots[itemID] -= move
		dst.InputSlots[itemID] += move
	}
}

// stepSplitter distributes items from src.OutputSlots round-robin across its outputs.
func stepSplitter(src *world.Building, buildings map[string]*world.Building, cap int64) {
	dests := make([]*world.Building, 0, len(src.Connections))
	for _, connID := range src.Connections {
		if d := realDest(connID, buildings); d != nil {
			dests = append(dests, d)
		}
	}
	if len(dests) == 0 {
		return
	}

	sharePerOutput := cap / int64(len(dests))
	if sharePerOutput == 0 {
		return
	}

	for itemID, qty := range src.OutputSlots {
		if qty == 0 {
			continue
		}
		for _, dest := range dests {
			space := InputBufCap - dest.InputSlots[itemID]
			if space <= 0 {
				continue
			}
			move := min64(qty, min64(sharePerOutput, space))
			if move <= 0 {
				continue
			}
			src.OutputSlots[itemID] -= move
			dest.InputSlots[itemID] += move
			qty -= move
		}
	}
}

// realDest follows a connection chain, skipping conveyor tiles, and returns the
// first non-conveyor building. Returns nil if the chain is broken or loops.
func realDest(startID string, buildings map[string]*world.Building) *world.Building {
	visited := make(map[string]bool)
	id := startID
	for {
		if visited[id] {
			return nil // cycle guard
		}
		visited[id] = true
		b, ok := buildings[id]
		if !ok {
			return nil
		}
		if b.Type != recipe.BuildingConveyor {
			return b
		}
		if len(b.Connections) == 0 {
			return nil // belt goes nowhere
		}
		id = b.Connections[0]
	}
}

// topoSort returns building IDs in topological order (sources first).
// Conveyors are included as transparent edges. Buildings in cycles are appended last.
func topoSort(buildings map[string]*world.Building) []string {
	// Build in-degree map over all buildings
	inDegree := make(map[string]int, len(buildings))
	for id := range buildings {
		inDegree[id] = 0
	}
	for _, b := range buildings {
		for _, conn := range b.Connections {
			if _, ok := buildings[conn]; ok {
				inDegree[conn]++
			}
		}
	}

	// Kahn's algorithm
	queue := make([]string, 0, len(buildings))
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]string, 0, len(buildings))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)
		for _, conn := range buildings[id].Connections {
			if _, ok := buildings[conn]; !ok {
				continue
			}
			inDegree[conn]--
			if inDegree[conn] == 0 {
				queue = append(queue, conn)
			}
		}
	}

	// Append any nodes not reached (cycle members) — process them anyway
	inResult := make(map[string]bool, len(result))
	for _, id := range result {
		inResult[id] = true
	}
	for id := range buildings {
		if !inResult[id] {
			result = append(result, id)
		}
	}

	return result
}
