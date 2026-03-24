What was added:


<li>static/research.json — 14-node research tree: iron/copper/stone/gold processing, basic automation (splitter/merger), basic assembly, circuits, oil refining, silicon, advanced assembly, microchips, belt speed I/II, extractor Mk.II/III</li>
<li>static/static.go — now also embeds research.json</li>
<li>internal/research/types.go — Node, WorldResearch (with BeltTier, MaxExtractorTier), NodeProgress</li>
<li>internal/research/registry.go — indexes nodes by ID and unlock token; IsBuildingUnlocked, DeriveWorldResearch</li>
<li>internal/research/repository.go — Firestore CRUD + UnlockTx (atomic read-validate-write)</li>
<li>internal/research/service.go — GetTree, GetWorldProgress, UnlockNode, IsBuildingUnlocked (satisfies world.ResearchChecker)</li>
<li>internal/research/handler.go — GET /research, GET /worlds/{id}/research, POST /worlds/{id}/research/{nodeID}/unlock</li>
<li>internal/world/service.go — ResearchChecker interface + SetResearchChecker</li>
<li>internal/world/building_service.go — ErrResearchLocked, research gate in PlaceBuilding</li>
<li>internal/tick/tick.go — BeltTierFn injection; belt tier now read from research state on every tick</li>

<br>
Starter buildings (extractor, conveyor, research lab) are always available. Everything else is gated behind the research tree. The unlock transaction is atomic — research points and coins are deducted in the same Firestore transaction as the unlock.