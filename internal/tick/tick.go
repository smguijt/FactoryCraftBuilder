// Package tick orchestrates the simulation cycle:
// load world state from Firestore → run pure simulation engine → persist mutations.
// It sits above both the world and simulation packages, breaking the import cycle.
package tick

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/simulation"
	"github.com/smguijt/factorycraftbuilder/internal/world"
)

// Orchestrator loads, simulates, and persists one tick for a world.
type Orchestrator struct {
	repo          *world.Repository
	registry      *recipe.Registry
	maxOfflineSec int64
	fs            *firestore.Client
}

func New(repo *world.Repository, registry *recipe.Registry, fs *firestore.Client, maxOfflineSec int64) *Orchestrator {
	return &Orchestrator{repo: repo, registry: registry, fs: fs, maxOfflineSec: maxOfflineSec}
}

// Run advances the simulation to now and returns the updated map snapshot.
// If the delta is too small to warrant a write, it returns the current snapshot unchanged.
func (o *Orchestrator) Run(ctx context.Context, playerID, worldID string) (*world.MapSnapshot, error) {
	// 1. Load current world state
	snap, err := o.repo.GetMapSnapshot(ctx, playerID, worldID)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	// 2. Build simulation WorldState (deep copy buildings so the engine can mutate freely)
	buildings := make(map[string]*world.Building, len(snap.Buildings))
	for _, b := range snap.Buildings {
		cp := *b
		cp.InputSlots = copyMap(b.InputSlots)
		cp.OutputSlots = copyMap(b.OutputSlots)
		cp.Connections = append([]string{}, b.Connections...)
		buildings[cp.ID] = &cp
	}

	invCopy := world.Inventory{
		Items:          copyMap(snap.Inventory.Items),
		TotalDelivered: copyMap(snap.Inventory.TotalDelivered),
		Coins:          snap.Inventory.Coins,
		ResearchPoints: snap.Inventory.ResearchPoints,
	}

	state := simulation.WorldState{
		World:     *snap.World,
		Buildings: buildings,
		Inventory: invCopy,
		Registry:  o.registry,
		BeltTier:  1, // Phase 5 will wire player belt tier from research
	}

	// 3. Run pure simulation
	result := simulation.Tick(state, time.Now().UTC(), o.maxOfflineSec)
	if result.Skipped {
		return snap, nil
	}

	// 4. Persist mutations
	if err := o.persist(ctx, playerID, worldID, snap.World, result); err != nil {
		return nil, fmt.Errorf("persist tick: %w", err)
	}

	// 5. Rebuild snapshot from result for the response
	updatedBuildings := make([]*world.Building, 0, len(result.MutatedBuildings))
	for _, b := range result.MutatedBuildings {
		updatedBuildings = append(updatedBuildings, b)
	}
	updatedInv := result.UpdatedInventory
	updatedWorld := *snap.World
	updatedWorld.LastSimulatedAt = result.NewLastSimulated

	return &world.MapSnapshot{
		World:         &updatedWorld,
		ResourceNodes: snap.ResourceNodes,
		Buildings:     updatedBuildings,
		Inventory:     &updatedInv,
	}, nil
}

// persist writes all mutated buildings + inventory + world timestamp.
// We use BulkWriter for buildings (fast, no atomicity needed per-building),
// and a transaction for inventory + world timestamp (these are financially sensitive).
func (o *Orchestrator) persist(
	ctx context.Context,
	playerID, worldID string,
	w *world.World,
	result simulation.SimulationResult,
) error {
	// Write mutated buildings via BulkWriter
	if err := o.repo.SaveBuildings(ctx, playerID, worldID, mapValues(result.MutatedBuildings)); err != nil {
		return fmt.Errorf("save buildings: %w", err)
	}

	// Atomically update inventory + world.lastSimulatedAt
	invRef := o.repo.InventoryRef(playerID, worldID)
	worldRef := o.repo.WorldRef(playerID, worldID)

	return o.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		inv := result.UpdatedInventory
		if err := tx.Set(invRef, &inv); err != nil {
			return err
		}
		return tx.Update(worldRef, []firestore.Update{
			{Path: "lastSimulatedAt", Value: result.NewLastSimulated},
		})
	})
}

func copyMap(m map[string]int64) map[string]int64 {
	if m == nil {
		return map[string]int64{}
	}
	cp := make(map[string]int64, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func mapValues(m map[string]*world.Building) []*world.Building {
	out := make([]*world.Building, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
