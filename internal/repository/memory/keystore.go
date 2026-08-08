package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
)

type KeyStore struct {
	mu     sync.RWMutex
	keys   map[string]*model.ApiKey
	byID   map[int64]string
	nextID int64
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		keys:   make(map[string]*model.ApiKey),
		byID:   make(map[int64]string),
		nextID: 1,
	}
}

func (s *KeyStore) Create(ctx context.Context, key *model.ApiKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[key.KeyHash]; exists {
		return model.ErrConflict
	}
	if key.Rank == "" {
		key.Rank = "member"
	}
	key.ID = s.nextID
	s.nextID++
	key.CreatedAt = time.Now()

	cp := *key
	cp.Scopes = append([]string(nil), key.Scopes...)
	s.keys[key.KeyHash] = &cp
	s.byID[cp.ID] = cp.KeyHash
	return nil
}

func (s *KeyStore) GetByHash(ctx context.Context, hash string) (*model.ApiKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.keys[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *key
	cp.Scopes = append([]string(nil), key.Scopes...)
	return &cp, nil
}

func (s *KeyStore) List(ctx context.Context) ([]model.ApiKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]model.ApiKey, 0, len(s.keys))
	for _, key := range s.keys {
		cp := *key
		cp.Scopes = append([]string(nil), key.Scopes...)
		keys = append(keys, cp)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	return keys, nil
}

func (s *KeyStore) Revoke(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash, ok := s.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	now := time.Now()
	s.keys[hash].RevokedAt = &now
	return nil
}

func (s *KeyStore) Touch(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash, ok := s.byID[id]
	if !ok {
		return model.ErrNotFound
	}
	now := time.Now()
	s.keys[hash].LastUsedAt = &now
	s.keys[hash].RequestCount++
	return nil
}
