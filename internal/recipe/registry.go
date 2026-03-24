package recipe

import (
	"encoding/json"
	"fmt"
)

// Registry holds all static game data loaded once at startup.
type Registry struct {
	ExtractionByNode map[ResourceNodeType]*ExtractionRecipe
	RecipeByID       map[string]*Recipe
	ItemByID         map[string]*Item
	BuildingByType   map[BuildingType]*BuildingDef
	Recipes          []*Recipe
}

type staticData struct {
	ExtractionRecipes []ExtractionRecipe `json:"extractionRecipes"`
	Recipes           []Recipe           `json:"recipes"`
	Items             []Item             `json:"items"`
	Buildings         []BuildingDef      `json:"buildings"`
}

// LoadRegistry parses the contents of recipes.json (passed as raw bytes).
// Call this once at startup (e.g. from main via go:embed).
func LoadRegistry(data []byte) (*Registry, error) {
	var sd staticData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse recipes.json: %w", err)
	}

	r := &Registry{
		ExtractionByNode: make(map[ResourceNodeType]*ExtractionRecipe, len(sd.ExtractionRecipes)),
		RecipeByID:       make(map[string]*Recipe, len(sd.Recipes)),
		ItemByID:         make(map[string]*Item, len(sd.Items)),
		BuildingByType:   make(map[BuildingType]*BuildingDef, len(sd.Buildings)),
	}

	for i := range sd.ExtractionRecipes {
		e := &sd.ExtractionRecipes[i]
		r.ExtractionByNode[e.NodeType] = e
	}
	for i := range sd.Recipes {
		rec := &sd.Recipes[i]
		r.RecipeByID[rec.ID] = rec
		r.Recipes = append(r.Recipes, rec)
	}
	for i := range sd.Items {
		item := &sd.Items[i]
		r.ItemByID[item.ID] = item
	}
	for i := range sd.Buildings {
		b := &sd.Buildings[i]
		r.BuildingByType[b.Type] = b
	}

	return r, nil
}
