package research

import (
	"time"

	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

// DeliveryRequirement is a cumulative delivery gate for a research node.
type DeliveryRequirement struct {
	ItemID   recipe.ItemID `json:"itemID"`
	Quantity int64         `json:"quantity"`
}

// Node is a static research tree node loaded from research.json at startup.
type Node struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Description          string                `json:"description"`
	DeliveryRequirements []DeliveryRequirement `json:"deliveryRequirements"`
	ResearchPointCost    int64                 `json:"researchPointCost"`
	CoinCost             int64                 `json:"coinCost"`
	Prerequisites        []string              `json:"prerequisites"` // node IDs
	// Unlocks is a list of buildingType strings, recipeIDs, or special tokens:
	//   "belt_tier_2", "belt_tier_3", "extractor_tier_2", "extractor_tier_3"
	Unlocks []string `json:"unlocks"`
}

// WorldResearch is the per-world research state document at worlds/{worldID}/research/state.
type WorldResearch struct {
	UnlockedNodes    []string  `json:"unlockedNodes" firestore:"unlockedNodes"`
	BeltTier         int       `json:"beltTier" firestore:"beltTier"`
	MaxExtractorTier int       `json:"maxExtractorTier" firestore:"maxExtractorTier"`
	UpdatedAt        time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// NodeProgress is returned by GET /worlds/{worldID}/research — combines static
// node definition with per-world progress for the frontend.
type NodeProgress struct {
	Node
	IsUnlocked       bool             `json:"isUnlocked"`
	DeliveryProgress map[string]int64 `json:"deliveryProgress"` // itemID → totalDelivered so far
	CanUnlock        bool             `json:"canUnlock"`        // all requirements satisfied
}
