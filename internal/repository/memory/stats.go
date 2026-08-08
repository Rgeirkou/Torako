package memory

import (
	"context"
	"sync"

	"github.com/rgeirkou/tyrako/internal/model"
)

// StatsStore is an in-memory stats store for single-instance deployments
// and tests. Use the PostgreSQL store for multi-instance deployments.
type StatsStore struct {
	mu            sync.Mutex
	twCount       int64
	twAmountCents int64
	twErrors      int64
	twRefs        map[string]struct{}
}

func NewStatsStore() *StatsStore {
	return &StatsStore{
		twRefs: make(map[string]struct{}),
	}
}

// RecordTw counts the redemption only if ref has not been counted before.
// An empty ref is always counted.
func (s *StatsStore) RecordTw(ctx context.Context, amountCents int64, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ref != "" {
		if _, seen := s.twRefs[ref]; seen {
			return nil
		}
		s.twRefs[ref] = struct{}{}
	}
	s.twCount++
	s.twAmountCents += amountCents
	return nil
}

// RecordTwError counts a failed redemption attempt. Errors are never
// deduplicated: each failed attempt is counted.
func (s *StatsStore) RecordTwError(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.twErrors++
	return nil
}

func (s *StatsStore) Totals(ctx context.Context) (*model.Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &model.Stats{
		Count:  s.twCount,
		Amount: float64(s.twAmountCents) / 100,
		Errors: s.twErrors,
		TrueMoney: model.StatsPart{
			Amount: float64(s.twAmountCents) / 100,
			Count:  s.twCount,
			Errors: s.twErrors,
		},
	}, nil
}
