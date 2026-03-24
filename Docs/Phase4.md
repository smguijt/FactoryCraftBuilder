<li>internal/simulation/engine.go — Tick pure function, WorldState/SimulationResult types</li>
<li>internal/simulation/extractor.go — fills output buffers based on rate × tier × delta, capped at 50</li>
<li>internal/simulation/conveyor.go — topological sort (Kahn's), belt-chain traversal, splitter round-robin, merger; isActive only gates production, not output drainage</li>
<li>internal/simulation/factory.go — cycle-based production for all factory types; research lab drains to inventory</li>
<li>internal/simulation/engine_test.go — 15 tests, 100% of tick logic covered</li>
<li>internal/tick/tick.go — orchestrator: load → copy → simulate → persist (BulkWriter for buildings, transaction for inventory + timestamp)</li>
<li>world.Handler now accepts a Ticker interface; both GET /map and POST /tick trigger the engine</li>