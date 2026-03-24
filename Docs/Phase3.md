<li>internal/world/building_service.go — PlaceBuilding (bounds + occupancy + extractor-on-node + funds tx), UpdateBuilding, DeleteBuilding, Connect (cycle detection + splitter/merger caps), Disconnect</li>
<li>internal/world/building_handler.go — all 7 building endpoints + GET /buildings/{buildingType}/recipes</li>
<li>internal/world/repository.go — BuildingLinkedToNode, CountIncomingConnections, RemoveFromAllConnections, PlaceBuildingTx</li>
<li>internal/world/handler.go — Tick stub for Phase 4</li>