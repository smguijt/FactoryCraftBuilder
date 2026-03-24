package player

import (
	"context"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Upsert creates or updates a player document on login.
// DisplayName is only updated if non-empty.
func (s *Service) Upsert(ctx context.Context, playerID, email, displayName string) (*Player, error) {
	existing, err := s.repo.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update mutable fields only
		if email != "" {
			existing.Email = email
		}
		if displayName != "" {
			existing.DisplayName = displayName
		}
		if err := s.repo.Upsert(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	p := &Player{
		ID:          playerID,
		Email:       email,
		DisplayName: displayName,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, playerID string) (*Player, error) {
	p, err := s.repo.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}
