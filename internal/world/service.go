package world

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

// ResearchChecker is implemented by research.Service. Defined here to avoid an import cycle.
type ResearchChecker interface {
	IsBuildingUnlocked(ctx context.Context, playerID, worldID string, bType recipe.BuildingType) (bool, error)
}

type Service struct {
	repo            *Repository
	registry        *recipe.Registry // exported via field for handlers that need recipe lookups
	startingCoins   int64
	researchChecker ResearchChecker // optional; nil = all buildings allowed (useful in tests)
}

// Registry exposes the loaded recipe registry (e.g. for building-type recipe filtering).
func (s *Service) Registry() *recipe.Registry { return s.registry }

// SetResearchChecker wires the research checker after construction to break the init cycle.
func (s *Service) SetResearchChecker(rc ResearchChecker) { s.researchChecker = rc }

func NewService(repo *Repository, registry *recipe.Registry, startingCoins int64) *Service {
	return &Service{repo: repo, registry: registry, startingCoins: startingCoins}
}

// CreateWorld generates a new world with resource nodes and a starting inventory.
func (s *Service) CreateWorld(ctx context.Context, playerID, name string) (*World, error) {
	now := time.Now().UTC()
	worldID := uuid.New().String()
	seed := now.UnixNano() ^ int64(len(name)*31)

	w := &World{
		ID:              worldID,
		PlayerID:        playerID,
		Name:            name,
		Seed:            seed,
		Width:           defaultWidth,
		Height:          defaultHeight,
		CreatedAt:       now,
		LastSimulatedAt: now,
	}

	nodes := GenerateNodes(worldID, seed, w.Width, w.Height)

	// Override inventory coins to starting amount
	if err := s.repo.CreateWorld(ctx, w, nodes); err != nil {
		return nil, fmt.Errorf("create world: %w", err)
	}

	// Set starting coins after world creation
	inv, err := s.repo.GetInventory(ctx, playerID, worldID)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}
	inv.Coins = s.startingCoins
	if err := s.repo.SaveInventory(ctx, playerID, worldID, inv); err != nil {
		return nil, fmt.Errorf("save inventory: %w", err)
	}

	return w, nil
}

func (s *Service) GetWorld(ctx context.Context, playerID, worldID string) (*World, error) {
	return s.repo.GetWorld(ctx, playerID, worldID)
}

func (s *Service) ListWorlds(ctx context.Context, playerID string) ([]*World, error) {
	return s.repo.ListWorlds(ctx, playerID)
}

func (s *Service) DeleteWorld(ctx context.Context, playerID, worldID string) error {
	if _, err := s.repo.GetWorld(ctx, playerID, worldID); err != nil {
		return err
	}
	return s.repo.DeleteWorld(ctx, playerID, worldID)
}

func (s *Service) GetMapSnapshot(ctx context.Context, playerID, worldID string) (*MapSnapshot, error) {
	return s.repo.GetMapSnapshot(ctx, playerID, worldID)
}

func (s *Service) ListNodes(ctx context.Context, playerID, worldID string) ([]*ResourceNode, error) {
	return s.repo.ListNodes(ctx, playerID, worldID)
}

func (s *Service) GetNode(ctx context.Context, playerID, worldID, nodeID string) (*ResourceNode, error) {
	return s.repo.GetNode(ctx, playerID, worldID, nodeID)
}

func (s *Service) GetInventory(ctx context.Context, playerID, worldID string) (*Inventory, error) {
	return s.repo.GetInventory(ctx, playerID, worldID)
}
