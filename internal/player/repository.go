package player

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Repository struct {
	fs *firestore.Client
}

func NewRepository(fs *firestore.Client) *Repository {
	return &Repository{fs: fs}
}

func (r *Repository) Get(ctx context.Context, playerID string) (*Player, error) {
	doc, err := r.fs.Collection("players").Doc(playerID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var p Player
	if err := doc.DataTo(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Upsert(ctx context.Context, p *Player) error {
	ref := r.fs.Collection("players").Doc(p.ID)
	_, err := ref.Set(ctx, p, firestore.MergeAll)
	return err
}

var ErrNotFound = errors.New("player not found")
