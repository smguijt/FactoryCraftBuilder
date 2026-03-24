package research

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNotFound = errors.New("research state not found")

type Repository struct {
	fs *firestore.Client
}

func NewRepository(fs *firestore.Client) *Repository {
	return &Repository{fs: fs}
}

func (r *Repository) stateRef(playerID, worldID string) *firestore.DocumentRef {
	return r.fs.Collection("players").Doc(playerID).
		Collection("worlds").Doc(worldID).
		Collection("research").Doc("state")
}

func (r *Repository) Get(ctx context.Context, playerID, worldID string) (*WorldResearch, error) {
	doc, err := r.stateRef(playerID, worldID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			// Return empty default — world starts with no unlocks
			return &WorldResearch{
				UnlockedNodes:    []string{},
				BeltTier:         1,
				MaxExtractorTier: 1,
				UpdatedAt:        time.Now().UTC(),
			}, nil
		}
		return nil, err
	}
	var wr WorldResearch
	return &wr, doc.DataTo(&wr)
}

func (r *Repository) Save(ctx context.Context, playerID, worldID string, wr *WorldResearch) error {
	_, err := r.stateRef(playerID, worldID).Set(ctx, wr)
	return err
}

// UnlockTx atomically validates and applies a research unlock against inventory.
// invRef is the inventory/state doc ref (from world.Repository.InventoryRef).
func (r *Repository) UnlockTx(
	ctx context.Context,
	playerID, worldID string,
	invRef *firestore.DocumentRef,
	applyFn func(tx *firestore.Transaction, wr *WorldResearch, rawInv map[string]any) error,
) error {
	stateRef := r.stateRef(playerID, worldID)

	return r.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Read research state
		doc, err := tx.Get(stateRef)
		var wr WorldResearch
		if err != nil {
			if status.Code(err) == codes.NotFound {
				wr = WorldResearch{
					UnlockedNodes:    []string{},
					BeltTier:         1,
					MaxExtractorTier: 1,
				}
			} else {
				return err
			}
		} else if err := doc.DataTo(&wr); err != nil {
			return err
		}

		// Read inventory as raw map so we can read/write it generically
		invDoc, err := tx.Get(invRef)
		if err != nil {
			return err
		}
		rawInv := invDoc.Data()

		// Delegate validation + mutation to caller
		if err := applyFn(tx, &wr, rawInv); err != nil {
			return err
		}

		// Persist both documents
		if err := tx.Set(stateRef, &wr); err != nil {
			return err
		}
		return tx.Set(invRef, rawInv)
	})
}
