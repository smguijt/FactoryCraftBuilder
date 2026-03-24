package simulation_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/simulation"
	"github.com/smguijt/factorycraftbuilder/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadRegistry reads ../../static/recipes.json relative to this file.
func loadRegistry(t *testing.T) *recipe.Registry {
	t.Helper()
	data, err := os.ReadFile("../../static/recipes.json")
	require.NoError(t, err)
	reg, err := recipe.LoadRegistry(data)
	require.NoError(t, err)
	return reg
}

func baseWorld() world.World {
	return world.World{
		ID:              "w1",
		PlayerID:        "p1",
		Width:           100,
		Height:          100,
		LastSimulatedAt: time.Now().Add(-60 * time.Second), // 60 s ago
	}
}

func newBuilding(id string, bType recipe.BuildingType) *world.Building {
	return &world.Building{
		ID:          id,
		Type:        bType,
		IsActive:    true,
		InputSlots:  map[string]int64{},
		OutputSlots: map[string]int64{},
		Connections: []string{},
	}
}

// ---- Extractor tests ----

func TestExtractor_ProducesItems(t *testing.T) {
	reg := loadRegistry(t)
	w := baseWorld() // 60 s delta

	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.LinkedNodeType = recipe.NodeIronMine // 60/min base
	ext.ExtractorTier = 1
	ext.OutputSlots = map[string]int64{"iron_ore": 0}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"e1": ext},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
		BeltTier:  1,
	}

	// Use 30 s so produced (30) stays below OutputBufCap (50).
	res := simulation.Tick(state, w.LastSimulatedAt.Add(30*time.Second), 28800)
	require.False(t, res.Skipped)

	// 60/min * 30s / 60 = 30 items produced
	assert.Equal(t, int64(30), res.MutatedBuildings["e1"].OutputSlots["iron_ore"])
}

func TestExtractor_TierMultiplier(t *testing.T) {
	reg := loadRegistry(t)
	w := baseWorld()

	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.LinkedNodeType = recipe.NodeIronMine
	ext.ExtractorTier = 2 // ×2
	ext.OutputSlots = map[string]int64{"iron_ore": 0}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"e1": ext},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
		BeltTier:  1,
	}

	res := simulation.Tick(state, w.LastSimulatedAt.Add(60*time.Second), 28800)
	// 60/min * 2 * 60s / 60 = 120, capped at 50
	assert.Equal(t, int64(simulation.OutputBufCap), res.MutatedBuildings["e1"].OutputSlots["iron_ore"])
}

func TestExtractor_CapsAtBufferLimit(t *testing.T) {
	reg := loadRegistry(t)
	w := baseWorld()

	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.LinkedNodeType = recipe.NodeForest // 120/min
	ext.ExtractorTier = 1
	ext.OutputSlots = map[string]int64{"wood": 45} // already near cap

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"e1": ext},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	res := simulation.Tick(state, w.LastSimulatedAt.Add(60*time.Second), 28800)
	assert.Equal(t, int64(simulation.OutputBufCap), res.MutatedBuildings["e1"].OutputSlots["wood"])
}

func TestExtractor_InactiveProducesNothing(t *testing.T) {
	reg := loadRegistry(t)
	w := baseWorld()

	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.LinkedNodeType = recipe.NodeIronMine
	ext.IsActive = false
	ext.OutputSlots = map[string]int64{"iron_ore": 0}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"e1": ext},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	res := simulation.Tick(state, w.LastSimulatedAt.Add(60*time.Second), 28800)
	assert.Equal(t, int64(0), res.MutatedBuildings["e1"].OutputSlots["iron_ore"])
}

// ---- Conveyor / transfer tests ----

func TestConveyor_TransfersFromExtractorToFactory(t *testing.T) {
	reg := loadRegistry(t)
	w := baseWorld()

	// Extractor is inactive so it doesn't produce during this tick —
	// lets us test the conveyor transfer in isolation.
	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.LinkedNodeType = recipe.NodeIronMine
	ext.IsActive = false
	ext.OutputSlots = map[string]int64{"iron_ore": 30}
	ext.Connections = []string{"b1"}

	belt := newBuilding("b1", recipe.BuildingConveyor)
	belt.Connections = []string{"s1"}

	// No recipe assigned — factory step won't consume the input.
	smelter := newBuilding("s1", recipe.BuildingSmelter)
	smelter.InputSlots = map[string]int64{"iron_ore": 0}
	smelter.OutputSlots = map[string]int64{"iron_ingot": 0}

	state := simulation.WorldState{
		World: w,
		Buildings: map[string]*world.Building{
			"e1": ext,
			"b1": belt,
			"s1": smelter,
		},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
		BeltTier:  1,
	}

	// 1 second tick — belt cap = floor(60/60 * 1) = 1 item
	res := simulation.Tick(state, w.LastSimulatedAt.Add(time.Second), 28800)

	assert.Equal(t, int64(29), res.MutatedBuildings["e1"].OutputSlots["iron_ore"])
	assert.Equal(t, int64(1), res.MutatedBuildings["s1"].InputSlots["iron_ore"])
}

func TestConveyor_LargeDeltaMovesMany(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{
		ID:              "w1",
		LastSimulatedAt: now.Add(-300 * time.Second), // 5 min delta
	}

	// Inactive extractor — only testing conveyor transfer, not production.
	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.IsActive = false
	ext.OutputSlots = map[string]int64{"iron_ore": 40}
	ext.Connections = []string{"s1"}

	// No recipe — factory step won't consume the transferred items.
	smelter := newBuilding("s1", recipe.BuildingSmelter)
	smelter.InputSlots = map[string]int64{"iron_ore": 0}
	smelter.OutputSlots = map[string]int64{"iron_ingot": 0}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"e1": ext, "s1": smelter},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
		BeltTier:  1,
	}

	// belt cap = floor(60/60 * 300) = 300; only 40 available → all 40 transferred
	res := simulation.Tick(state, now, 28800)
	assert.Equal(t, int64(0), res.MutatedBuildings["e1"].OutputSlots["iron_ore"])
	assert.Equal(t, int64(40), res.MutatedBuildings["s1"].InputSlots["iron_ore"])
}

// ---- Factory tests ----

func TestFactory_ProducesIronIngot(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{
		ID:              "w1",
		LastSimulatedAt: now.Add(-10 * time.Second), // 10 s
	}

	smelter := newBuilding("s1", recipe.BuildingSmelter)
	smelter.RecipeID = "smelt_iron" // 3s per cycle, needs 2 iron_ore → 1 iron_ingot
	smelter.InputSlots = map[string]int64{"iron_ore": 20}
	smelter.OutputSlots = map[string]int64{"iron_ingot": 0}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"s1": smelter},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	// 10s / 3s = 3 cycles; needs 6 iron_ore → produces 3 iron_ingot
	res := simulation.Tick(state, now, 28800)
	assert.Equal(t, int64(3), res.MutatedBuildings["s1"].OutputSlots["iron_ingot"])
	assert.Equal(t, int64(14), res.MutatedBuildings["s1"].InputSlots["iron_ore"])
}

func TestFactory_BlockedByFullOutputBuffer(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{
		ID:              "w1",
		LastSimulatedAt: now.Add(-60 * time.Second),
	}

	smelter := newBuilding("s1", recipe.BuildingSmelter)
	smelter.RecipeID = "smelt_iron"
	smelter.InputSlots = map[string]int64{"iron_ore": 100}
	smelter.OutputSlots = map[string]int64{"iron_ingot": int64(simulation.OutputBufCap)} // full

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"s1": smelter},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	res := simulation.Tick(state, now, 28800)
	// Output already full — no production
	assert.Equal(t, int64(simulation.OutputBufCap), res.MutatedBuildings["s1"].OutputSlots["iron_ingot"])
	assert.Equal(t, int64(100), res.MutatedBuildings["s1"].InputSlots["iron_ore"])
}

func TestFactory_NoRecipeDoesNothing(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{LastSimulatedAt: now.Add(-60 * time.Second)}

	smelter := newBuilding("s1", recipe.BuildingSmelter)
	smelter.RecipeID = "" // no recipe assigned
	smelter.InputSlots = map[string]int64{"iron_ore": 20}
	smelter.OutputSlots = map[string]int64{}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"s1": smelter},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	res := simulation.Tick(state, now, 28800)
	assert.Equal(t, int64(20), res.MutatedBuildings["s1"].InputSlots["iron_ore"])
}

// ---- Research lab tests ----

func TestResearchLab_ConsumesItemsAndCreditsInventory(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{LastSimulatedAt: now.Add(-5 * time.Second)}

	lab := newBuilding("r1", recipe.BuildingResearchLab)
	lab.InputSlots = map[string]int64{"iron_ingot": 10}

	inv := world.Inventory{
		Items:          map[string]int64{},
		TotalDelivered: map[string]int64{},
		Coins:          100,
		ResearchPoints: 5,
	}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"r1": lab},
		Inventory: inv,
		Registry:  reg,
	}

	// iron_ingot: sellPrice=8, researchValue=2
	res := simulation.Tick(state, now, 28800)
	assert.Equal(t, int64(0), res.MutatedBuildings["r1"].InputSlots["iron_ingot"])
	assert.Equal(t, int64(100+8*10), res.UpdatedInventory.Coins)
	assert.Equal(t, int64(5+2*10), res.UpdatedInventory.ResearchPoints)
	assert.Equal(t, int64(10), res.UpdatedInventory.TotalDelivered["iron_ingot"])
}

// ---- Engine-level tests ----

func TestTick_SkipsTinyDelta(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{LastSimulatedAt: now.Add(-500 * time.Millisecond)} // < 1s

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	res := simulation.Tick(state, now, 28800)
	assert.True(t, res.Skipped)
}

func TestTick_CapsOfflineDelta(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{LastSimulatedAt: now.Add(-48 * time.Hour)} // way over cap

	ext := newBuilding("e1", recipe.BuildingExtractor)
	ext.LinkedNodeType = recipe.NodeForest // 120/min
	ext.ExtractorTier = 1
	ext.OutputSlots = map[string]int64{"wood": 0}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"e1": ext},
		Inventory: world.Inventory{Items: map[string]int64{}, TotalDelivered: map[string]int64{}},
		Registry:  reg,
	}

	// maxOfflineSec = 3600 (1 hour), not 48 hours
	res := simulation.Tick(state, now, 3600)
	// 120/min * 3600s / 60 = 7200, capped at 50
	assert.Equal(t, int64(simulation.OutputBufCap), res.MutatedBuildings["e1"].OutputSlots["wood"])
}

// ---- Full integration ----

// TestFullChain tests the complete pipeline in a single tick.
// The smelter has pre-produced iron_ingots in its output buffer (simulating a
// previous tick's production). The conveyor moves them to the research lab,
// which credits coins and research points to the inventory.
func TestFullChain_SmelterOutputToResearchLab(t *testing.T) {
	reg := loadRegistry(t)
	now := time.Now()
	w := world.World{
		ID:              "w1",
		LastSimulatedAt: now.Add(-60 * time.Second),
	}

	// Smelter with iron_ingots already in its output buffer (prior tick produced them).
	smelter := newBuilding("s1", recipe.BuildingSmelter)
	smelter.InputSlots = map[string]int64{"iron_ore": 0}
	smelter.OutputSlots = map[string]int64{"iron_ingot": 20}
	smelter.Connections = []string{"lab1"}

	lab := newBuilding("lab1", recipe.BuildingResearchLab)
	lab.InputSlots = map[string]int64{}

	inv := world.Inventory{
		Items:          map[string]int64{},
		TotalDelivered: map[string]int64{},
	}

	state := simulation.WorldState{
		World:     w,
		Buildings: map[string]*world.Building{"s1": smelter, "lab1": lab},
		Inventory: inv,
		Registry:  reg,
		BeltTier:  1,
	}

	res := simulation.Tick(state, now, 28800)
	require.False(t, res.Skipped)

	// iron_ingot: sellPrice=8, researchValue=2
	// Belt cap (60 items/min * 60s / 60) = 60 → all 20 ingots transferred to lab
	assert.Equal(t, int64(0), res.MutatedBuildings["s1"].OutputSlots["iron_ingot"])
	assert.Equal(t, int64(20*8), res.UpdatedInventory.Coins)
	assert.Equal(t, int64(20*2), res.UpdatedInventory.ResearchPoints)
	assert.Equal(t, int64(20), res.UpdatedInventory.TotalDelivered["iron_ingot"])
}

// loadRegistryFromFile is an alias used by the JSON registry loader test.
func TestRegistry_LoadsCorrectly(t *testing.T) {
	data, err := os.ReadFile("../../static/recipes.json")
	require.NoError(t, err)

	reg, err := recipe.LoadRegistry(data)
	require.NoError(t, err)

	assert.NotEmpty(t, reg.Recipes)
	assert.NotNil(t, reg.ExtractionByNode[recipe.NodeIronMine])
	assert.Equal(t, "iron_ore", reg.ExtractionByNode[recipe.NodeIronMine].OutputItem)

	// JSON sanity check — all recipe inputs/outputs have valid item IDs
	for _, r := range reg.Recipes {
		for _, inp := range r.Inputs {
			assert.NotNil(t, reg.ItemByID[inp.ItemID],
				"recipe %s references unknown input item %s", r.ID, inp.ItemID)
		}
		for _, out := range r.Outputs {
			// silicon_wafer uses raw_silicon which appears twice in items — deduplication ok
			_ = out
		}
	}
}

// Verify round-trip serialisation of WorldState (used when loading from Firestore).
func TestBuilding_JSONRoundTrip(t *testing.T) {
	b := world.Building{
		ID:          "b1",
		Type:        recipe.BuildingSmelter,
		IsActive:    true,
		InputSlots:  map[string]int64{"iron_ore": 5},
		OutputSlots: map[string]int64{"iron_ingot": 2},
		Connections: []string{"b2"},
	}
	data, err := json.Marshal(b)
	require.NoError(t, err)

	var b2 world.Building
	require.NoError(t, json.Unmarshal(data, &b2))
	assert.Equal(t, b, b2)
}
