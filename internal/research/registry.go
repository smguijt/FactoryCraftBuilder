package research

import (
	"encoding/json"
	"fmt"

	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

// Registry indexes the static research tree loaded from research.json at startup.
type Registry struct {
	Nodes   []*Node
	ByID    map[string]*Node
	// ByUnlock maps an unlock token (buildingType, recipeID, or special) to the node that grants it.
	ByUnlock map[string]*Node
}

// LoadRegistry parses research.json bytes into a Registry.
func LoadRegistry(data []byte) (*Registry, error) {
	var nodes []*Node
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("parse research.json: %w", err)
	}
	r := &Registry{
		Nodes:    nodes,
		ByID:     make(map[string]*Node, len(nodes)),
		ByUnlock: make(map[string]*Node),
	}
	for _, n := range nodes {
		r.ByID[n.ID] = n
		for _, u := range n.Unlocks {
			r.ByUnlock[u] = n
		}
	}
	return r, nil
}

// starterBuildings are available without any research unlock.
var starterBuildings = map[recipe.BuildingType]bool{
	recipe.BuildingExtractor:   true,
	recipe.BuildingConveyor:    true,
	recipe.BuildingResearchLab: true,
}

// IsBuildingLocked returns true if placing this building type requires research.
func (r *Registry) IsBuildingLocked(bType recipe.BuildingType) bool {
	if starterBuildings[bType] {
		return false
	}
	_, gated := r.ByUnlock[string(bType)]
	return gated
}

// IsBuildingUnlocked returns true if the building type is either a starter building
// or has been unlocked by one of the provided node IDs.
func (r *Registry) IsBuildingUnlocked(bType recipe.BuildingType, unlockedNodeIDs []string) bool {
	if starterBuildings[bType] {
		return true
	}
	node, gated := r.ByUnlock[string(bType)]
	if !gated {
		// Not gated by research — always available (shouldn't happen with our data, but safe).
		return true
	}
	for _, id := range unlockedNodeIDs {
		if id == node.ID {
			return true
		}
	}
	return false
}

// DeriveWorldResearch computes the derived fields of WorldResearch from a set of unlocked node IDs.
// Call this after any unlock to keep beltTier and maxExtractorTier in sync.
func (r *Registry) DeriveWorldResearch(unlockedIDs []string) (beltTier, maxExtractorTier int) {
	beltTier = 1
	maxExtractorTier = 1
	unlocked := make(map[string]bool, len(unlockedIDs))
	for _, id := range unlockedIDs {
		unlocked[id] = true
	}
	for _, id := range unlockedIDs {
		node := r.ByID[id]
		if node == nil {
			continue
		}
		for _, u := range node.Unlocks {
			switch u {
			case "belt_tier_2":
				if beltTier < 2 {
					beltTier = 2
				}
			case "belt_tier_3":
				if beltTier < 3 {
					beltTier = 3
				}
			case "extractor_tier_2":
				if maxExtractorTier < 2 {
					maxExtractorTier = 2
				}
			case "extractor_tier_3":
				if maxExtractorTier < 3 {
					maxExtractorTier = 3
				}
			}
		}
		_ = unlocked
	}
	return
}
