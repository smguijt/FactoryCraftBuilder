package simulation

import (
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/world"
)

var factoryTypes = map[recipe.BuildingType]bool{
	recipe.BuildingSmelter:       true,
	recipe.BuildingAssembler:     true,
	recipe.BuildingAdvAssembler:  true,
	recipe.BuildingChemicalPlant: true,
}

// stepFactories runs production cycles for all active factory buildings.
func stepFactories(buildings map[string]*world.Building, reg *recipe.Registry, delta float64) {
	for _, b := range buildings {
		if !b.IsActive || b.RecipeID == "" {
			continue
		}
		if !factoryTypes[b.Type] {
			continue
		}

		rec, ok := reg.RecipeByID[b.RecipeID]
		if !ok || rec.CraftingTimeSec <= 0 {
			continue
		}

		// Maximum cycles possible given elapsed time
		cyclesCap := ifloor(delta / rec.CraftingTimeSec)
		if cyclesCap == 0 {
			continue
		}

		// Cap by available inputs
		for _, inp := range rec.Inputs {
			have := b.InputSlots[inp.ItemID]
			cyclesCap = min64(cyclesCap, have/int64(inp.Quantity))
		}
		if cyclesCap == 0 {
			continue
		}

		// Cap by available output space
		for _, out := range rec.Outputs {
			space := int64(OutputBufCap) - b.OutputSlots[out.ItemID]
			cyclesCap = min64(cyclesCap, space/int64(out.Quantity))
		}
		if cyclesCap == 0 {
			continue
		}

		// Apply production
		for _, inp := range rec.Inputs {
			b.InputSlots[inp.ItemID] -= int64(inp.Quantity) * cyclesCap
		}
		for _, out := range rec.Outputs {
			b.OutputSlots[out.ItemID] += int64(out.Quantity) * cyclesCap
		}
	}
}

// stepResearchLabs consumes items from research lab input buffers and credits
// coins + research points to the world inventory. Updates totalDelivered.
func stepResearchLabs(buildings map[string]*world.Building, reg *recipe.Registry, inv *world.Inventory) {
	if inv.Items == nil {
		inv.Items = map[string]int64{}
	}
	if inv.TotalDelivered == nil {
		inv.TotalDelivered = map[string]int64{}
	}

	for _, b := range buildings {
		if b.Type != recipe.BuildingResearchLab || !b.IsActive {
			continue
		}
		for itemID, qty := range b.InputSlots {
			if qty == 0 {
				continue
			}
			item, ok := reg.ItemByID[itemID]
			if !ok {
				continue
			}
			inv.Coins += item.SellPrice * qty
			inv.ResearchPoints += item.ResearchValue * qty
			inv.TotalDelivered[itemID] += qty
			b.InputSlots[itemID] = 0
		}
	}
}
