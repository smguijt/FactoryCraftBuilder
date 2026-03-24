package world

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
)

const (
	defaultWidth       = 100
	defaultHeight      = 100
	minNodeDistance    = 3  // minimum Chebyshev distance between any two resource nodes
	outputBufferCap    = 50 // default output buffer capacity for buildings
)

// nodeWeight controls how frequently each node type appears.
// Higher weight = more nodes of that type on the map.
var nodeWeights = []struct {
	nodeType recipe.ResourceNodeType
	weight   int
}{
	{recipe.NodeForest, 30},
	{recipe.NodeIronMine, 20},
	{recipe.NodeCopperMine, 20},
	{recipe.NodeStoneMine, 15},
	{recipe.NodeCoalMine, 15},
	{recipe.NodeGoldMine, 8},
	{recipe.NodeOilWell, 6},
	{recipe.NodeSiliconMine, 6},
	{recipe.NodeUranium, 3},
	{recipe.NodeWolframite, 3},
}

// totalNodes is how many resource nodes to place on a default 100×100 map.
// Scales with map area if a non-default size is used.
const baseNodeDensity = 300 // nodes per 10000 tiles

// GenerateNodes creates a deterministic set of resource nodes for a world.
// The seed from the world document ensures reproducibility.
func GenerateNodes(worldID string, seed int64, width, height int) []*ResourceNode {
	rng := rand.New(rand.NewSource(seed))

	total := baseNodeDensity * width * height / 10000
	if total < 50 {
		total = 50
	}

	// Build cumulative weight table for O(1) weighted sampling.
	cumWeights := make([]int, len(nodeWeights))
	sum := 0
	for i, nw := range nodeWeights {
		sum += nw.weight
		cumWeights[i] = sum
	}

	placed := make(map[[2]int]bool)
	nodes := make([]*ResourceNode, 0, total)

	attempts := 0
	for len(nodes) < total && attempts < total*20 {
		attempts++
		x := rng.Intn(width)
		y := rng.Intn(height)

		if !isFarEnough(x, y, placed, minNodeDistance) {
			continue
		}

		nodeType := weightedPick(rng, cumWeights, sum)
		placed[[2]int{x, y}] = true

		nodes = append(nodes, &ResourceNode{
			ID:      uuid.New().String(),
			WorldID: worldID,
			Type:    nodeType,
			X:       x,
			Y:       y,
		})
	}

	return nodes
}

// isFarEnough returns true if (x,y) is at least minDist away (Chebyshev) from all placed nodes.
func isFarEnough(x, y int, placed map[[2]int]bool, minDist int) bool {
	for pos := range placed {
		dx := x - pos[0]
		if dx < 0 {
			dx = -dx
		}
		dy := y - pos[1]
		if dy < 0 {
			dy = -dy
		}
		// Chebyshev distance = max(|dx|, |dy|)
		d := dx
		if dy > d {
			d = dy
		}
		if d < minDist {
			return false
		}
	}
	return true
}

// weightedPick picks a node type using a pre-built cumulative weight table.
func weightedPick(rng *rand.Rand, cumWeights []int, total int) recipe.ResourceNodeType {
	r := rng.Intn(total)
	for i, cw := range cumWeights {
		if r < cw {
			return nodeWeights[i].nodeType
		}
	}
	return nodeWeights[0].nodeType
}
