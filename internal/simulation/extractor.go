package simulation

import (
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/world"
)

// stepExtractors fills each active extractor's output buffer based on delta seconds.
func stepExtractors(buildings map[string]*world.Building, reg *recipe.Registry, delta float64) {
	for _, b := range buildings {
		if b.Type != recipe.BuildingExtractor || !b.IsActive {
			continue
		}

		ext, ok := reg.ExtractionByNode[b.LinkedNodeType]
		if !ok {
			continue // misconfigured extractor (no node type); skip
		}

		mult := tierMultipliers[b.ExtractorTier]
		if mult == 0 {
			mult = 1.0
		}

		// items produced = floor(rate_per_sec * delta)
		ratePerSec := ext.RatePerMin * mult / 60.0
		produced := ifloor(ratePerSec * delta)
		if produced == 0 {
			continue
		}

		current := b.OutputSlots[ext.OutputItem]
		b.OutputSlots[ext.OutputItem] = min64(current+produced, OutputBufCap)
	}
}
