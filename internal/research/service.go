package research

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

var (
	ErrAlreadyUnlocked      = errors.New("research node already unlocked")
	ErrPrereqNotMet         = errors.New("prerequisite research nodes not yet unlocked")
	ErrDeliveryNotMet       = errors.New("delivery requirements not met")
	ErrInsufficientRP       = errors.New("not enough research points")
	ErrInsufficientCoins    = errors.New("not enough coins")
	ErrUnknownNode          = errors.New("unknown research node")
)

type Service struct {
	repo     *Repository
	registry *Registry
	invRefFn func(playerID, worldID string) *firestore.DocumentRef
}

func NewService(repo *Repository, registry *Registry, invRefFn func(playerID, worldID string) *firestore.DocumentRef) *Service {
	return &Service{repo: repo, registry: registry, invRefFn: invRefFn}
}

// GetTree returns the full static research tree.
func (s *Service) GetTree() []*Node {
	return s.registry.Nodes
}

// GetWorldProgress returns each node with per-world unlock state and delivery progress.
func (s *Service) GetWorldProgress(ctx context.Context, playerID, worldID string, totalDelivered map[string]int64) ([]*NodeProgress, error) {
	wr, err := s.repo.Get(ctx, playerID, worldID)
	if err != nil {
		return nil, err
	}

	unlockedSet := make(map[string]bool, len(wr.UnlockedNodes))
	for _, id := range wr.UnlockedNodes {
		unlockedSet[id] = true
	}

	result := make([]*NodeProgress, 0, len(s.registry.Nodes))
	for _, node := range s.registry.Nodes {
		np := &NodeProgress{
			Node:             *node,
			IsUnlocked:       unlockedSet[node.ID],
			DeliveryProgress: make(map[string]int64, len(node.DeliveryRequirements)),
		}

		// Delivery progress
		deliveryMet := true
		for _, req := range node.DeliveryRequirements {
			have := totalDelivered[req.ItemID]
			np.DeliveryProgress[req.ItemID] = have
			if have < req.Quantity {
				deliveryMet = false
			}
		}

		// Prerequisites
		prereqMet := true
		for _, prereqID := range node.Prerequisites {
			if !unlockedSet[prereqID] {
				prereqMet = false
				break
			}
		}

		np.CanUnlock = !np.IsUnlocked && deliveryMet && prereqMet
		result = append(result, np)
	}
	return result, nil
}

// UnlockNode validates all conditions and atomically unlocks a research node.
func (s *Service) UnlockNode(ctx context.Context, playerID, worldID, nodeID string) (*WorldResearch, error) {
	node, ok := s.registry.ByID[nodeID]
	if !ok {
		return nil, ErrUnknownNode
	}

	invRef := s.invRefFn(playerID, worldID)
	var result *WorldResearch

	err := s.repo.UnlockTx(ctx, playerID, worldID, invRef,
		func(tx *firestore.Transaction, wr *WorldResearch, rawInv map[string]any) error {
			// Already unlocked?
			for _, id := range wr.UnlockedNodes {
				if id == nodeID {
					return ErrAlreadyUnlocked
				}
			}

			// Prerequisites
			unlockedSet := make(map[string]bool, len(wr.UnlockedNodes))
			for _, id := range wr.UnlockedNodes {
				unlockedSet[id] = true
			}
			for _, prereq := range node.Prerequisites {
				if !unlockedSet[prereq] {
					return fmt.Errorf("%w: %s", ErrPrereqNotMet, prereq)
				}
			}

			// Read inventory fields
			coins := asInt64(rawInv["coins"])
			rp := asInt64(rawInv["researchPoints"])
			totalDelivered := asInt64Map(rawInv["totalDelivered"])

			// Delivery requirements
			for _, req := range node.DeliveryRequirements {
				if totalDelivered[req.ItemID] < req.Quantity {
					return fmt.Errorf("%w: need %d %s, have %d", ErrDeliveryNotMet,
						req.Quantity, req.ItemID, totalDelivered[req.ItemID])
				}
			}

			// Research points
			if rp < node.ResearchPointCost {
				return fmt.Errorf("%w: need %d, have %d", ErrInsufficientRP, node.ResearchPointCost, rp)
			}

			// Coins
			if coins < node.CoinCost {
				return fmt.Errorf("%w: need %d, have %d", ErrInsufficientCoins, node.CoinCost, coins)
			}

			// Apply unlock
			wr.UnlockedNodes = append(wr.UnlockedNodes, nodeID)
			wr.BeltTier, wr.MaxExtractorTier = s.registry.DeriveWorldResearch(wr.UnlockedNodes)
			wr.UpdatedAt = time.Now().UTC()

			// Deduct costs
			rawInv["researchPoints"] = rp - node.ResearchPointCost
			rawInv["coins"] = coins - node.CoinCost

			result = wr
			return nil
		},
	)
	return result, err
}

// GetState returns the raw WorldResearch for a world (used by tick orchestrator).
func (s *Service) GetState(ctx context.Context, playerID, worldID string) (*WorldResearch, error) {
	return s.repo.Get(ctx, playerID, worldID)
}

// IsBuildingUnlocked satisfies world.ResearchChecker.
func (s *Service) IsBuildingUnlocked(ctx context.Context, playerID, worldID string, bType recipe.BuildingType) (bool, error) {
	wr, err := s.repo.Get(ctx, playerID, worldID)
	if err != nil {
		return false, err
	}
	return s.registry.IsBuildingUnlocked(bType, wr.UnlockedNodes), nil
}

// MaxExtractorTier satisfies world.ResearchChecker.
func (s *Service) MaxExtractorTier(ctx context.Context, playerID, worldID string) (int, error) {
	wr, err := s.repo.Get(ctx, playerID, worldID)
	if err != nil {
		return 1, err
	}
	if wr.MaxExtractorTier < 1 {
		return 1, nil
	}
	return wr.MaxExtractorTier, nil
}

// helpers for reading Firestore map[string]any values

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case float64:
		return int64(t)
	case int:
		return int64(t)
	}
	return 0
}

func asInt64Map(v any) map[string]int64 {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(m))
	for k, val := range m {
		out[k] = asInt64(val)
	}
	return out
}
