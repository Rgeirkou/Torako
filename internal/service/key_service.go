package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/rgeirkou/tyrako/internal/apikey"
	"github.com/rgeirkou/tyrako/internal/model"
)

var allowedScopes = map[string]bool{
	"tw":    true,
	"admin": true,
}

var allowedRanks = map[string]bool{
	"member":  true,
	"partner": true,
	"admin":   true,
}

const defaultRank = "member"

type apiKeyRepository interface {
	Create(ctx context.Context, key *model.ApiKey) error
	GetByHash(ctx context.Context, hash string) (*model.ApiKey, error)
	List(ctx context.Context) ([]model.ApiKey, error)
	Revoke(ctx context.Context, id int64) error
	Touch(ctx context.Context, id int64) error
}

type KeyService struct {
	repo apiKeyRepository
}

func NewKeyService(repo apiKeyRepository) *KeyService {
	return &KeyService{repo: repo}
}

func (s *KeyService) CreateKey(ctx context.Context, in model.CreateApiKeyInput) (model.ApiKeyView, error) {
	if in.Name == "" || len(in.Name) > 64 {
		return model.ApiKeyView{}, &model.ValidationError{Details: model.FieldErrors{
			{Field: "name", Message: "required (max 64 chars)"},
		}}
	}
	if len(in.Scopes) == 0 {
		return model.ApiKeyView{}, &model.ValidationError{Details: model.FieldErrors{
			{Field: "scopes", Message: "at least one scope is required"},
		}}
	}
	for _, scope := range in.Scopes {
		if !allowedScopes[scope] {
			return model.ApiKeyView{}, &model.ValidationError{Details: model.FieldErrors{
				{Field: "scopes", Message: fmt.Sprintf("invalid scope %q (allowed: tw, admin)", scope)},
			}}
		}
	}
	rank := in.Rank
	if rank == "" {
		rank = defaultRank
	}
	if !allowedRanks[rank] {
		return model.ApiKeyView{}, &model.ValidationError{Details: model.FieldErrors{
			{Field: "rank", Message: fmt.Sprintf("invalid rank %q (allowed: member, partner, admin)", rank)},
		}}
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now()) {
		return model.ApiKeyView{}, &model.ValidationError{Details: model.FieldErrors{
			{Field: "expires_at", Message: "must be in the future"},
		}}
	}

	plaintext, err := apikey.Generate()
	if err != nil {
		return model.ApiKeyView{}, fmt.Errorf("create api key: %w", err)
	}

	key := &model.ApiKey{
		Name:      in.Name,
		Rank:      rank,
		KeyHash:   apikey.Hash(plaintext),
		Scopes:    append([]string(nil), in.Scopes...),
		ExpiresAt: in.ExpiresAt,
	}
	if err := s.repo.Create(ctx, key); err != nil {
		return model.ApiKeyView{}, fmt.Errorf("create api key: %w", err)
	}
	return viewFrom(key, plaintext), nil
}

func (s *KeyService) ListKeys(ctx context.Context) ([]model.ApiKeyView, error) {
	keys, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	views := make([]model.ApiKeyView, 0, len(keys))
	for _, key := range keys {
		cp := key
		views = append(views, viewFrom(&cp, ""))
	}
	return views, nil
}

func (s *KeyService) RevokeKey(ctx context.Context, id int64) error {
	if err := s.repo.Revoke(ctx, id); err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	return nil
}

func (s *KeyService) RotateKey(ctx context.Context, id int64) (model.ApiKeyView, error) {
	keys, err := s.repo.List(ctx)
	if err != nil {
		return model.ApiKeyView{}, fmt.Errorf("rotate api key: %w", err)
	}
	var current *model.ApiKey
	for i := range keys {
		if keys[i].ID == id {
			current = &keys[i]
			break
		}
	}
	if current == nil {
		return model.ApiKeyView{}, fmt.Errorf("rotate api key: %w", model.ErrNotFound)
	}
	// Create the replacement first: if it fails the old key stays usable.
	view, err := s.CreateKey(ctx, model.CreateApiKeyInput{
		Name:      current.Name,
		Rank:      current.Rank,
		Scopes:    current.Scopes,
		ExpiresAt: current.ExpiresAt,
	})
	if err != nil {
		return model.ApiKeyView{}, fmt.Errorf("rotate api key: %w", err)
	}
	if err := s.repo.Revoke(ctx, id); err != nil {
		return model.ApiKeyView{}, fmt.Errorf("rotate api key: %w", err)
	}
	return view, nil
}

// Bootstrap registers a seed key. With a fixed BOOTSTRAP_API_KEY the key is
// registered idempotently (skipped if already present). With an empty
// BOOTSTRAP_API_KEY it only generates a random key when the store is empty.
func (s *KeyService) Bootstrap(ctx context.Context, plaintext string, logger *slog.Logger) error {
	if plaintext == "" {
		keys, err := s.repo.List(ctx)
		if err != nil {
			return fmt.Errorf("bootstrap api key: %w", err)
		}
		if len(keys) > 0 {
			return nil
		}
		generated, err := apikey.Generate()
		if err != nil {
			return fmt.Errorf("bootstrap api key: %w", err)
		}
		plaintext = generated
		logger.Warn("generated bootstrap api key for empty store", "hint", "set BOOTSTRAP_API_KEY to use a fixed key", "key", plaintext)
	} else {
		if _, err := s.repo.GetByHash(ctx, apikey.Hash(plaintext)); err == nil {
			return nil
		}
		logger.Info("registering bootstrap api key")
	}

	key := &model.ApiKey{
		Name:    "bootstrap",
		Rank:    "admin",
		KeyHash: apikey.Hash(plaintext),
		Scopes:  []string{"tw", "admin"},
	}
	if err := s.repo.Create(ctx, key); err != nil && !errors.Is(err, model.ErrConflict) {
		return fmt.Errorf("bootstrap api key: %w", err)
	}
	return nil
}

func viewFrom(key *model.ApiKey, plaintext string) model.ApiKeyView {
	return model.ApiKeyView{
		ID:           key.ID,
		Name:         key.Name,
		Rank:         key.Rank,
		Key:          plaintext,
		Scopes:       key.Scopes,
		ExpiresAt:    key.ExpiresAt,
		RevokedAt:    key.RevokedAt,
		LastUsedAt:   key.LastUsedAt,
		RequestCount: key.RequestCount,
		CreatedAt:    key.CreatedAt,
	}
}
