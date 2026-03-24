package debug

import (
	"fmt"
	"strings"

	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/internal/world"
)

const tileSize = 10 // SVG pixels per map tile

// nodeColors maps resource node types to fill colours.
var nodeColors = map[recipe.ResourceNodeType]string{
	recipe.NodeForest:      "#2d6a2d",
	recipe.NodeIronMine:    "#8c8c8c",
	recipe.NodeCopperMine:  "#b5651d",
	recipe.NodeStoneMine:   "#a0a0a0",
	recipe.NodeGoldMine:    "#d4af37",
	recipe.NodeCoalMine:    "#2b2b2b",
	recipe.NodeOilWell:     "#3a3a5c",
	recipe.NodeSiliconMine: "#c0a060",
	recipe.NodeUranium:     "#5c8c3a",
	recipe.NodeWolframite:  "#6a4c8c",
}

// nodeLabels maps node types to single-character labels.
var nodeLabels = map[recipe.ResourceNodeType]string{
	recipe.NodeForest:      "F",
	recipe.NodeIronMine:    "I",
	recipe.NodeCopperMine:  "C",
	recipe.NodeStoneMine:   "S",
	recipe.NodeGoldMine:    "G",
	recipe.NodeCoalMine:    "Co",
	recipe.NodeOilWell:     "O",
	recipe.NodeSiliconMine: "Si",
	recipe.NodeUranium:     "U",
	recipe.NodeWolframite:  "W",
}

var buildingColors = map[recipe.BuildingType]string{
	recipe.BuildingExtractor:     "#e07030",
	recipe.BuildingConveyor:      "#f0d060",
	recipe.BuildingSplitter:      "#f0a020",
	recipe.BuildingMerger:        "#d08020",
	recipe.BuildingSmelter:       "#e04040",
	recipe.BuildingAssembler:     "#4080e0",
	recipe.BuildingAdvAssembler:  "#2050c0",
	recipe.BuildingChemicalPlant: "#40c080",
	recipe.BuildingResearchLab:   "#c040c0",
}

// GenerateSVG renders a map snapshot as an SVG string.
func GenerateSVG(snap *world.MapSnapshot) string {
	w := snap.World.Width
	h := snap.World.Height
	svgW := w * tileSize
	svgH := h * tileSize

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" style="background:#1a1a1a">`, svgW, svgH)

	// Resource nodes
	for _, n := range snap.ResourceNodes {
		color := nodeColors[n.Type]
		if color == "" {
			color = "#555"
		}
		label := nodeLabels[n.Type]
		if label == "" {
			label = "?"
		}
		x := n.X * tileSize
		y := n.Y * tileSize
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s" rx="1"/>`,
			x, y, tileSize, tileSize, color)
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="5" fill="white" text-anchor="middle" dominant-baseline="middle">%s</text>`,
			x+tileSize/2, y+tileSize/2, label)
	}

	// Belt connection lines
	buildingPos := make(map[string][2]int, len(snap.Buildings))
	for _, bld := range snap.Buildings {
		buildingPos[bld.ID] = [2]int{bld.X, bld.Y}
	}
	for _, bld := range snap.Buildings {
		for _, conn := range bld.Connections {
			if dst, ok := buildingPos[conn]; ok {
				x1 := bld.X*tileSize + tileSize/2
				y1 := bld.Y*tileSize + tileSize/2
				x2 := dst[0]*tileSize + tileSize/2
				y2 := dst[1]*tileSize + tileSize/2
				fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#ffff00" stroke-width="0.5" opacity="0.6"/>`,
					x1, y1, x2, y2)
			}
		}
	}

	// Buildings (drawn on top of connections)
	for _, bld := range snap.Buildings {
		color := buildingColors[bld.Type]
		if color == "" {
			color = "#888"
		}
		x := bld.X * tileSize
		y := bld.Y * tileSize
		// Slightly smaller than a tile so resource nodes show through when overlapping
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s" rx="2" opacity="0.9"/>`,
			x+1, y+1, tileSize-2, tileSize-2, color)

		// Rotation arrow
		arrowSVG := rotationArrow(bld.Rotation, x, y)
		b.WriteString(arrowSVG)
	}

	b.WriteString(`</svg>`)
	return b.String()
}

// rotationArrow returns a small directional triangle for a building tile.
func rotationArrow(rotation, x, y int) string {
	cx := x + tileSize/2
	cy := y + tileSize/2
	half := tileSize/2 - 2

	// tip and two base corners depending on rotation
	var pts string
	switch rotation {
	case 0: // North
		pts = fmt.Sprintf("%d,%d %d,%d %d,%d", cx, cy-half, cx-2, cy+2, cx+2, cy+2)
	case 90: // East
		pts = fmt.Sprintf("%d,%d %d,%d %d,%d", cx+half, cy, cx-2, cy-2, cx-2, cy+2)
	case 180: // South
		pts = fmt.Sprintf("%d,%d %d,%d %d,%d", cx, cy+half, cx-2, cy-2, cx+2, cy-2)
	default: // West (270)
		pts = fmt.Sprintf("%d,%d %d,%d %d,%d", cx-half, cy, cx+2, cy-2, cx+2, cy+2)
	}
	return fmt.Sprintf(`<polygon points="%s" fill="white" opacity="0.7"/>`, pts)
}
