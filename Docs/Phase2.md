static/recipes.json — extraction rates, 12 recipes, 24 items, building placement costs
static/static.go — embeds recipes.json at compile time
internal/recipe/types.go — all core domain types
internal/recipe/registry.go — parses and indexes the JSON at startup
internal/world/world.go — World, ResourceNode, Building, Inventory, MapSnapshot
internal/world/generator.go — seeded RNG, weighted node placement, Chebyshev min-distance enforcement (~300 nodes on a 100×100 map)
internal/world/repository.go — full Firestore CRUD for all sub-collections
internal/world/service.go — business logic + CreateWorld
internal/world/handler.go — GET/POST /worlds, GET/DELETE /worlds/{id}, /map, /nodes, /inventory