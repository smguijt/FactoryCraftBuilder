<li>static/recipes.json — extraction rates, 12 recipes, 24 items, building placement costs</li>
<li>static/static.go — embeds recipes.json at compile time</li>
<li>internal/recipe/types.go — all core domain types</li>
<li>internal/recipe/registry.go — parses and indexes the JSON at startup</li>
<li>internal/world/world.go — World, ResourceNode, Building, Inventory, MapSnapshot</li>
<li>internal/world/generator.go — seeded RNG, weighted node placement, Chebyshev min-distance enforcement </(~300 nodes on a 100×100 map)</li>
<li>internal/world/repository.go — full Firestore CRUD for all sub-collections</li>
<li>internal/world/service.go — business logic + CreateWorld</li>
<li>internal/world/handler.go — GET/POST /worlds, GET/DELETE /worlds/{id}, /map, /nodes, /inventory</li>