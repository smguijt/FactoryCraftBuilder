package world

import (
	"time"

	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

// World is the top-level world document stored at players/{playerID}/worlds/{worldID}.
type World struct {
	ID              string    `json:"id" firestore:"id"`
	PlayerID        string    `json:"playerID" firestore:"playerID"`
	Name            string    `json:"name" firestore:"name"`
	Seed            int64     `json:"seed" firestore:"seed"`
	Width           int       `json:"width" firestore:"width"`
	Height          int       `json:"height" firestore:"height"`
	CreatedAt       time.Time `json:"createdAt" firestore:"createdAt"`
	LastSimulatedAt time.Time `json:"lastSimulatedAt" firestore:"lastSimulatedAt"`
}

// ResourceNode is a resource node document in worlds/{worldID}/resourceNodes/{nodeID}.
type ResourceNode struct {
	ID       string                   `json:"id" firestore:"id"`
	WorldID  string                   `json:"worldID" firestore:"worldID"`
	Type     recipe.ResourceNodeType  `json:"type" firestore:"type"`
	X        int                      `json:"x" firestore:"x"`
	Y        int                      `json:"y" firestore:"y"`
}

// Building is a placed building document in worlds/{worldID}/buildings/{buildingID}.
type Building struct {
	ID           string               `json:"id" firestore:"id"`
	WorldID      string               `json:"worldID" firestore:"worldID"`
	Type         recipe.BuildingType  `json:"type" firestore:"type"`
	X            int                  `json:"x" firestore:"x"`
	Y            int                  `json:"y" firestore:"y"`
	Rotation     int                  `json:"rotation" firestore:"rotation"` // 0, 90, 180, 270
	RecipeID     string               `json:"recipeID,omitempty" firestore:"recipeID,omitempty"`
	LinkedNodeID string               `json:"linkedNodeID,omitempty" firestore:"linkedNodeID,omitempty"`
	// ExtractorTier is 1/2/3; only relevant for extractors.
	ExtractorTier int                  `json:"extractorTier" firestore:"extractorTier"`
	InputSlots    map[string]int64     `json:"inputSlots" firestore:"inputSlots"`
	OutputSlots   map[string]int64     `json:"outputSlots" firestore:"outputSlots"`
	Connections   []string             `json:"connections" firestore:"connections"` // ordered output building IDs
	IsActive      bool                 `json:"isActive" firestore:"isActive"`
	LastTickAt    time.Time            `json:"lastTickAt" firestore:"lastTickAt"`
}

// Inventory is stored at worlds/{worldID}/inventory/state.
type Inventory struct {
	Items          map[string]int64  `json:"items" firestore:"items"`
	Coins          int64             `json:"coins" firestore:"coins"`
	ResearchPoints int64             `json:"researchPoints" firestore:"researchPoints"`
	TotalDelivered map[string]int64  `json:"totalDelivered" firestore:"totalDelivered"`
}

// MapSnapshot is the full response for GET /worlds/{worldID}/map.
type MapSnapshot struct {
	World         *World          `json:"world"`
	ResourceNodes []*ResourceNode `json:"resourceNodes"`
	Buildings     []*Building     `json:"buildings"`
	Inventory     *Inventory      `json:"inventory"`
}
