package static

import _ "embed"

//go:embed recipes.json
var RecipesJSON []byte

//go:embed research.json
var ResearchJSON []byte
