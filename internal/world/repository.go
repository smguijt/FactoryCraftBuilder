package world

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrNotFound is returned when a document doesn't exist.
var ErrNotFound = errors.New("not found")

// Repository handles all Firestore operations for worlds and their sub-collections.
// Data lives under players/{playerID}/worlds/{worldID}/...
type Repository struct {
	fs *firestore.Client
}

func NewRepository(fs *firestore.Client) *Repository {
	return &Repository{fs: fs}
}

func (r *Repository) worldRef(playerID, worldID string) *firestore.DocumentRef {
	return r.fs.Collection("players").Doc(playerID).
		Collection("worlds").Doc(worldID)
}

// ---- World ----

func (r *Repository) CreateWorld(ctx context.Context, w *World, nodes []*ResourceNode) error {
	batch := r.fs.BulkWriter(ctx)

	wRef := r.worldRef(w.PlayerID, w.ID)
	if _, err := batch.Set(wRef, w); err != nil {
		return err
	}

	// Inventory
	inv := &Inventory{
		Items:          map[string]int64{},
		TotalDelivered: map[string]int64{},
	}
	invRef := wRef.Collection("inventory").Doc("state")
	if _, err := batch.Set(invRef, inv); err != nil {
		return err
	}

	// Resource nodes
	nodesCol := wRef.Collection("resourceNodes")
	for _, n := range nodes {
		ref := nodesCol.Doc(n.ID)
		if _, err := batch.Set(ref, n); err != nil {
			return err
		}
	}

	batch.End()
	return nil
}

func (r *Repository) GetWorld(ctx context.Context, playerID, worldID string) (*World, error) {
	doc, err := r.worldRef(playerID, worldID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var w World
	return &w, doc.DataTo(&w)
}

func (r *Repository) ListWorlds(ctx context.Context, playerID string) ([]*World, error) {
	iter := r.fs.Collection("players").Doc(playerID).Collection("worlds").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	worlds := make([]*World, 0, len(docs))
	for _, doc := range docs {
		var w World
		if err := doc.DataTo(&w); err != nil {
			return nil, err
		}
		worlds = append(worlds, &w)
	}
	return worlds, nil
}

func (r *Repository) DeleteWorld(ctx context.Context, playerID, worldID string) error {
	// Firestore doesn't cascade-delete subcollections; Cloud Functions or a batch job handles that.
	// For now we just delete the world document; subcollections become orphaned but harmless.
	_, err := r.worldRef(playerID, worldID).Delete(ctx)
	return err
}

// ---- Resource Nodes ----

func (r *Repository) ListNodes(ctx context.Context, playerID, worldID string) ([]*ResourceNode, error) {
	iter := r.worldRef(playerID, worldID).Collection("resourceNodes").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	nodes := make([]*ResourceNode, 0, len(docs))
	for _, doc := range docs {
		var n ResourceNode
		if err := doc.DataTo(&n); err != nil {
			return nil, err
		}
		nodes = append(nodes, &n)
	}
	return nodes, nil
}

func (r *Repository) GetNode(ctx context.Context, playerID, worldID, nodeID string) (*ResourceNode, error) {
	doc, err := r.worldRef(playerID, worldID).Collection("resourceNodes").Doc(nodeID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var n ResourceNode
	return &n, doc.DataTo(&n)
}

// ---- Buildings ----

func (r *Repository) ListBuildings(ctx context.Context, playerID, worldID string) ([]*Building, error) {
	iter := r.worldRef(playerID, worldID).Collection("buildings").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	buildings := make([]*Building, 0, len(docs))
	for _, doc := range docs {
		var b Building
		if err := doc.DataTo(&b); err != nil {
			return nil, err
		}
		buildings = append(buildings, &b)
	}
	return buildings, nil
}

func (r *Repository) GetBuilding(ctx context.Context, playerID, worldID, buildingID string) (*Building, error) {
	doc, err := r.worldRef(playerID, worldID).Collection("buildings").Doc(buildingID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var b Building
	return &b, doc.DataTo(&b)
}

func (r *Repository) SaveBuilding(ctx context.Context, playerID, worldID string, b *Building) error {
	ref := r.worldRef(playerID, worldID).Collection("buildings").Doc(b.ID)
	_, err := ref.Set(ctx, b)
	return err
}

func (r *Repository) DeleteBuilding(ctx context.Context, playerID, worldID, buildingID string) error {
	_, err := r.worldRef(playerID, worldID).Collection("buildings").Doc(buildingID).Delete(ctx)
	return err
}

// ---- Inventory ----

func (r *Repository) GetInventory(ctx context.Context, playerID, worldID string) (*Inventory, error) {
	doc, err := r.worldRef(playerID, worldID).Collection("inventory").Doc("state").Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var inv Inventory
	return &inv, doc.DataTo(&inv)
}

func (r *Repository) SaveInventory(ctx context.Context, playerID, worldID string, inv *Inventory) error {
	ref := r.worldRef(playerID, worldID).Collection("inventory").Doc("state")
	_, err := ref.Set(ctx, inv)
	return err
}

// ---- Map snapshot (all sub-collections in parallel) ----

func (r *Repository) GetMapSnapshot(ctx context.Context, playerID, worldID string) (*MapSnapshot, error) {
	world, err := r.GetWorld(ctx, playerID, worldID)
	if err != nil {
		return nil, err
	}

	nodes, err := r.ListNodes(ctx, playerID, worldID)
	if err != nil {
		return nil, err
	}

	buildings, err := r.ListBuildings(ctx, playerID, worldID)
	if err != nil {
		return nil, err
	}

	inv, err := r.GetInventory(ctx, playerID, worldID)
	if err != nil {
		return nil, err
	}

	return &MapSnapshot{
		World:         world,
		ResourceNodes: nodes,
		Buildings:     buildings,
		Inventory:     inv,
	}, nil
}

// SaveBuildings writes multiple buildings in a single BulkWriter batch.
func (r *Repository) SaveBuildings(ctx context.Context, playerID, worldID string, buildings []*Building) error {
	bw := r.fs.BulkWriter(ctx)
	col := r.worldRef(playerID, worldID).Collection("buildings")
	for _, b := range buildings {
		ref := col.Doc(b.ID)
		if _, err := bw.Set(ref, b); err != nil {
			return err
		}
	}
	bw.End()
	return nil
}

// UpdateWorldSimulatedAt updates only the lastSimulatedAt field.
func (r *Repository) UpdateWorldSimulatedAt(ctx context.Context, w *World) error {
	_, err := r.worldRef(w.PlayerID, w.ID).Update(ctx, []firestore.Update{
		{Path: "lastSimulatedAt", Value: w.LastSimulatedAt},
	})
	return err
}

// NodeAtPosition returns the resource node at (x,y) in a world, or nil if none.
func (r *Repository) NodeAtPosition(ctx context.Context, playerID, worldID string, x, y int) (*ResourceNode, error) {
	iter := r.worldRef(playerID, worldID).Collection("resourceNodes").
		Where("x", "==", x).
		Where("y", "==", y).
		Limit(1).
		Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	var n ResourceNode
	return &n, docs[0].DataTo(&n)
}

// BuildingAtPosition returns the building at (x,y) in a world, or nil if none.
func (r *Repository) BuildingAtPosition(ctx context.Context, playerID, worldID string, x, y int) (*Building, error) {
	iter := r.worldRef(playerID, worldID).Collection("buildings").
		Where("x", "==", x).
		Where("y", "==", y).
		Limit(1).
		Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	var b Building
	return &b, docs[0].DataTo(&b)
}
