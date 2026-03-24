package recipe

// ResourceNodeType is the type of a resource node on the map.
type ResourceNodeType string

const (
	NodeForest      ResourceNodeType = "forest"
	NodeIronMine    ResourceNodeType = "iron_mine"
	NodeCopperMine  ResourceNodeType = "copper_mine"
	NodeStoneMine   ResourceNodeType = "stone_mine"
	NodeGoldMine    ResourceNodeType = "gold_mine"
	NodeCoalMine    ResourceNodeType = "coal_mine"
	NodeUranium     ResourceNodeType = "uranium"
	NodeWolframite  ResourceNodeType = "wolframite"
	NodeOilWell     ResourceNodeType = "oil_well"
	NodeSiliconMine ResourceNodeType = "silicon_mine"
)

// BuildingType is the type of a placeable building.
type BuildingType string

const (
	BuildingExtractor        BuildingType = "extractor"
	BuildingConveyor         BuildingType = "conveyor"
	BuildingSplitter         BuildingType = "splitter"
	BuildingMerger           BuildingType = "merger"
	BuildingSmelter          BuildingType = "smelter"
	BuildingAssembler        BuildingType = "assembler"
	BuildingAdvAssembler     BuildingType = "advanced_assembler"
	BuildingChemicalPlant    BuildingType = "chemical_plant"
	BuildingResearchLab      BuildingType = "research_lab"
)

// ItemID is a string identifier for an item.
type ItemID = string

// RecipeInput is one ingredient in a recipe.
type RecipeInput struct {
	ItemID   ItemID `json:"itemID"`
	Quantity int    `json:"quantity"`
}

// RecipeOutput is one product (primary or byproduct) of a recipe.
type RecipeOutput struct {
	ItemID    ItemID `json:"itemID"`
	Quantity  int    `json:"quantity"`
	IsPrimary bool   `json:"isPrimary"`
}

// Recipe represents a crafting recipe.
type Recipe struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	FactoryType     BuildingType  `json:"factoryType"`
	CraftingTimeSec float64       `json:"craftingTimeSec"`
	Inputs          []RecipeInput `json:"inputs"`
	Outputs         []RecipeOutput `json:"outputs"`
}

// ExtractionRecipe describes what an extractor produces on a given node type.
type ExtractionRecipe struct {
	NodeType    ResourceNodeType `json:"nodeType"`
	OutputItem  ItemID           `json:"outputItem"`
	RatePerMin  float64          `json:"ratePerMin"` // base rate items/minute
}

// Item defines an item with economy values.
type Item struct {
	ID            ItemID  `json:"id"`
	Name          string  `json:"name"`
	SellPrice     int64   `json:"sellPrice"`     // coins per item delivered to Research Lab
	ResearchValue int64   `json:"researchValue"` // RP per item delivered
	BuyPrice      int64   `json:"buyPrice"`      // used for building cost calculations
}

// BuildingDef defines static properties of a building type.
type BuildingDef struct {
	Type          BuildingType   `json:"type"`
	PlacementCost PlacementCost  `json:"placementCost"`
}

// PlacementCost is what it costs to place a building.
type PlacementCost struct {
	Coins int64              `json:"coins"`
	Items []PlacementItem    `json:"items,omitempty"`
}

// PlacementItem is an item requirement for placement.
type PlacementItem struct {
	ItemID   ItemID `json:"itemID"`
	Quantity int    `json:"quantity"`
}
