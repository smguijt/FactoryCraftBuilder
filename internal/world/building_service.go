package world

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

var (
	ErrOccupied          = errors.New("tile already occupied")
	ErrNoNodeHere        = errors.New("extractor must be placed on a resource node")
	ErrNodeTaken         = errors.New("a building is already linked to that node")
	ErrOutOfBounds       = errors.New("position is outside world boundaries")
	ErrInvalidRotation   = errors.New("rotation must be 0, 90, 180 or 270")
	ErrInsufficientFunds = errors.New("insufficient coins or items")
	ErrUnknownBuilding   = errors.New("unknown building type")
	ErrUnknownRecipe     = errors.New("unknown recipe")
	ErrWrongFactory      = errors.New("recipe cannot be used in this building type")
	ErrCycleDetected     = errors.New("connection would create a cycle")
	ErrTooManyOutputs    = errors.New("splitter already has 3 output connections")
	ErrTooManyInputs     = errors.New("merger target already has 3 incoming connections")
	ErrAlreadyConnected  = errors.New("buildings are already connected")
	ErrSelfConnect       = errors.New("cannot connect a building to itself")
	ErrConnNotFound      = errors.New("connection not found")
)

// PlaceBuildingRequest is the input for placing a building.
type PlaceBuildingRequest struct {
	Type     recipe.BuildingType `json:"type"`
	X        int                 `json:"x"`
	Y        int                 `json:"y"`
	Rotation int                 `json:"rotation"`
}

// UpdateBuildingRequest is the input for patching a building.
type UpdateBuildingRequest struct {
	RecipeID *string `json:"recipeID"` // nil = no change
	IsActive *bool   `json:"isActive"` // nil = no change
}

// PlaceBuilding validates placement rules and deducts costs, then creates the building.
func (s *Service) PlaceBuilding(ctx context.Context, playerID, worldID string, req PlaceBuildingRequest) (*Building, error) {
	// Validate rotation
	if req.Rotation != 0 && req.Rotation != 90 && req.Rotation != 180 && req.Rotation != 270 {
		return nil, ErrInvalidRotation
	}

	// Validate building type
	def, ok := s.registry.BuildingByType[req.Type]
	if !ok {
		return nil, ErrUnknownBuilding
	}

	// Load world to check bounds
	w, err := s.repo.GetWorld(ctx, playerID, worldID)
	if err != nil {
		return nil, err
	}
	if req.X < 0 || req.X >= w.Width || req.Y < 0 || req.Y >= w.Height {
		return nil, ErrOutOfBounds
	}

	// Check tile occupancy
	existing, err := s.repo.BuildingAtPosition(ctx, playerID, worldID, req.X, req.Y)
	if err != nil {
		return nil, fmt.Errorf("check occupancy: %w", err)
	}
	if existing != nil {
		return nil, ErrOccupied
	}

	// Extractor-specific checks
	var linkedNodeID string
	var linkedNodeType recipe.ResourceNodeType
	if req.Type == recipe.BuildingExtractor {
		node, err := s.repo.NodeAtPosition(ctx, playerID, worldID, req.X, req.Y)
		if err != nil {
			return nil, fmt.Errorf("check node: %w", err)
		}
		if node == nil {
			return nil, ErrNoNodeHere
		}
		// Ensure no other extractor is already linked to this node
		linked, err := s.repo.BuildingLinkedToNode(ctx, playerID, worldID, node.ID)
		if err != nil {
			return nil, fmt.Errorf("check linked extractor: %w", err)
		}
		if linked != nil {
			return nil, ErrNodeTaken
		}
		linkedNodeID = node.ID
		linkedNodeType = node.Type
	}

	// Build the document
	buildingID := uuid.New().String()
	now := time.Now().UTC()

	b := &Building{
		ID:             buildingID,
		WorldID:        worldID,
		Type:           req.Type,
		X:              req.X,
		Y:              req.Y,
		Rotation:       req.Rotation,
		LinkedNodeID:   linkedNodeID,
		LinkedNodeType: linkedNodeType,
		ExtractorTier:  1,
		InputSlots:   map[string]int64{},
		OutputSlots:  initOutputSlots(s.registry, req.Type, linkedNodeType),
		Connections:  []string{},
		IsActive:     true,
		LastTickAt:   now,
	}

	// Deduct placement cost in a Firestore transaction
	if err := s.repo.PlaceBuildingTx(ctx, playerID, worldID, b, def.PlacementCost); err != nil {
		return nil, err
	}

	return b, nil
}

// initOutputSlots pre-creates the output slot key for extractors so the simulation
// engine doesn't need to look it up separately.
func initOutputSlots(reg *recipe.Registry, bType recipe.BuildingType, nodeType recipe.ResourceNodeType) map[string]int64 {
	if bType == recipe.BuildingExtractor && nodeType != "" {
		if ext, ok := reg.ExtractionByNode[nodeType]; ok {
			return map[string]int64{ext.OutputItem: 0}
		}
	}
	return map[string]int64{}
}

// ListBuildings returns all buildings in a world.
func (s *Service) ListBuildings(ctx context.Context, playerID, worldID string) ([]*Building, error) {
	return s.repo.ListBuildings(ctx, playerID, worldID)
}

// GetBuilding returns a single building.
func (s *Service) GetBuilding(ctx context.Context, playerID, worldID, buildingID string) (*Building, error) {
	return s.repo.GetBuilding(ctx, playerID, worldID, buildingID)
}

// UpdateBuilding patches a building's recipe and/or active state.
func (s *Service) UpdateBuilding(ctx context.Context, playerID, worldID, buildingID string, req UpdateBuildingRequest) (*Building, error) {
	b, err := s.repo.GetBuilding(ctx, playerID, worldID, buildingID)
	if err != nil {
		return nil, err
	}

	if req.RecipeID != nil {
		if *req.RecipeID == "" {
			b.RecipeID = ""
		} else {
			rec, ok := s.registry.RecipeByID[*req.RecipeID]
			if !ok {
				return nil, ErrUnknownRecipe
			}
			if rec.FactoryType != b.Type {
				return nil, ErrWrongFactory
			}
			b.RecipeID = *req.RecipeID
			// Pre-populate input/output slots for the new recipe
			b.InputSlots = make(map[string]int64, len(rec.Inputs))
			for _, inp := range rec.Inputs {
				b.InputSlots[inp.ItemID] = 0
			}
			b.OutputSlots = make(map[string]int64, len(rec.Outputs))
			for _, out := range rec.Outputs {
				b.OutputSlots[out.ItemID] = 0
			}
		}
	}

	if req.IsActive != nil {
		b.IsActive = *req.IsActive
	}

	if err := s.repo.SaveBuilding(ctx, playerID, worldID, b); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteBuilding removes a building and cleans up any incoming connections.
func (s *Service) DeleteBuilding(ctx context.Context, playerID, worldID, buildingID string) error {
	if _, err := s.repo.GetBuilding(ctx, playerID, worldID, buildingID); err != nil {
		return err
	}
	// Remove this building from any other building's connections list
	if err := s.repo.RemoveFromAllConnections(ctx, playerID, worldID, buildingID); err != nil {
		return fmt.Errorf("clean connections: %w", err)
	}
	return s.repo.DeleteBuilding(ctx, playerID, worldID, buildingID)
}

// Connect adds a directed connection from buildingID → targetID.
func (s *Service) Connect(ctx context.Context, playerID, worldID, buildingID, targetID string) error {
	if buildingID == targetID {
		return ErrSelfConnect
	}

	src, err := s.repo.GetBuilding(ctx, playerID, worldID, buildingID)
	if err != nil {
		return err
	}
	tgt, err := s.repo.GetBuilding(ctx, playerID, worldID, targetID)
	if err != nil {
		return err
	}

	// Already connected?
	for _, c := range src.Connections {
		if c == targetID {
			return ErrAlreadyConnected
		}
	}

	// Splitter output cap
	if src.Type == recipe.BuildingSplitter && len(src.Connections) >= 3 {
		return ErrTooManyOutputs
	}

	// Merger input cap: count how many buildings already connect TO targetID
	if tgt.Type == recipe.BuildingMerger {
		inCount, err := s.repo.CountIncomingConnections(ctx, playerID, worldID, targetID)
		if err != nil {
			return fmt.Errorf("count inputs: %w", err)
		}
		if inCount >= 3 {
			return ErrTooManyInputs
		}
	}

	// Cycle detection: DFS from target — if we can reach source, connecting would cycle
	all, err := s.repo.ListBuildings(ctx, playerID, worldID)
	if err != nil {
		return fmt.Errorf("load buildings for cycle check: %w", err)
	}
	if hasCycle(buildingID, targetID, all) {
		return ErrCycleDetected
	}

	src.Connections = append(src.Connections, targetID)
	return s.repo.SaveBuilding(ctx, playerID, worldID, src)
}

// Disconnect removes the connection from buildingID → targetID.
func (s *Service) Disconnect(ctx context.Context, playerID, worldID, buildingID, targetID string) error {
	src, err := s.repo.GetBuilding(ctx, playerID, worldID, buildingID)
	if err != nil {
		return err
	}

	idx := -1
	for i, c := range src.Connections {
		if c == targetID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrConnNotFound
	}

	src.Connections = append(src.Connections[:idx], src.Connections[idx+1:]...)
	return s.repo.SaveBuilding(ctx, playerID, worldID, src)
}

// hasCycle returns true if connecting src→tgt would create a cycle.
// It does a DFS from tgt following existing connections; if it reaches src, there's a cycle.
func hasCycle(srcID, tgtID string, all []*Building) bool {
	adj := make(map[string][]string, len(all))
	for _, b := range all {
		adj[b.ID] = b.Connections
	}

	visited := make(map[string]bool)
	var dfs func(id string) bool
	dfs = func(id string) bool {
		if id == srcID {
			return true
		}
		if visited[id] {
			return false
		}
		visited[id] = true
		for _, next := range adj[id] {
			if dfs(next) {
				return true
			}
		}
		return false
	}
	return dfs(tgtID)
}

// InventoryTxFn is a helper used by PlaceBuildingTx to validate and deduct placement cost.
func validateAndDeduct(inv *Inventory, cost recipe.PlacementCost) error {
	if inv.Coins < cost.Coins {
		return ErrInsufficientFunds
	}
	for _, item := range cost.Items {
		if inv.Items[item.ItemID] < int64(item.Quantity) {
			return ErrInsufficientFunds
		}
	}
	inv.Coins -= cost.Coins
	for _, item := range cost.Items {
		inv.Items[item.ItemID] -= int64(item.Quantity)
	}
	return nil
}

// PlaceBuildingTx wraps building creation + inventory deduction in a Firestore transaction.
func (r *Repository) PlaceBuildingTx(ctx context.Context, playerID, worldID string, b *Building, cost recipe.PlacementCost) error {
	invRef := r.worldRef(playerID, worldID).Collection("inventory").Doc("state")
	bRef := r.worldRef(playerID, worldID).Collection("buildings").Doc(b.ID)

	return r.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Read inventory
		doc, err := tx.Get(invRef)
		if err != nil {
			return fmt.Errorf("read inventory: %w", err)
		}
		var inv Inventory
		if err := doc.DataTo(&inv); err != nil {
			return err
		}
		if inv.Items == nil {
			inv.Items = map[string]int64{}
		}

		// Validate + deduct
		if err := validateAndDeduct(&inv, cost); err != nil {
			return err
		}

		if err := tx.Set(invRef, &inv); err != nil {
			return err
		}
		return tx.Set(bRef, b)
	})
}
