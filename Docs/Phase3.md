internal/world/building_service.go — PlaceBuilding (bounds + occupancy + extractor-on-node + funds tx), UpdateBuilding, DeleteBuilding, Connect (cycle detection + splitter/merger caps), Disconnect
internal/world/building_handler.go — all 7 building endpoints + GET /buildings/{buildingType}/recipes
internal/world/repository.go — BuildingLinkedToNode, CountIncomingConnections, RemoveFromAllConnections, PlaceBuildingTx
internal/world/handler.go — Tick stub for Phase 4