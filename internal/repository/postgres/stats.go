package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgeirkou/tyrako/internal/model"
)

// StatsStore persists the all-time success totals in a single row updated
// with atomic increments, so multi-instance deployments stay consistent.
type StatsStore struct {
	pool *pgxpool.Pool
}

func NewStatsStore(pool *pgxpool.Pool) *StatsStore {
	return &StatsStore{pool: pool}
}

// RecordTw counts the redemption only if ref has not been counted before.
// An empty ref is always counted.
func (s *StatsStore) RecordTw(ctx context.Context, amountCents int64, ref string) error {
	if err := s.countOnce(ctx, "tw", ref, `UPDATE stats_totals SET tw_count = tw_count + 1, tw_amount_cents = tw_amount_cents + $1 WHERE id = 1`, amountCents); err != nil {
		return fmt.Errorf("record tw stat: %w", err)
	}
	return nil
}

// countOnce inserts ref into recorded_refs and increments the totals only
// when the insert wins the race, all inside one transaction, so concurrent
// duplicates of the same ref still count exactly once.
func (s *StatsStore) countOnce(ctx context.Context, channel, ref, updateSQL string, amountCents int64) error {
	if ref == "" {
		_, err := s.pool.Exec(ctx, updateSQL, amountCents)
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `INSERT INTO recorded_refs (ref, channel) VALUES ($1, $2) ON CONFLICT (ref) DO NOTHING`, ref, channel)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, updateSQL, amountCents); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RecordTwError counts a failed redemption attempt. Errors are never
// deduplicated: each failed attempt is counted.
func (s *StatsStore) RecordTwError(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `UPDATE stats_totals SET tw_error_count = tw_error_count + 1 WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("record tw error stat: %w", err)
	}
	return nil
}

func (s *StatsStore) Totals(ctx context.Context) (*model.Stats, error) {
	var twCount, twCents, twErrors int64
	err := s.pool.QueryRow(ctx,
		`SELECT tw_count, tw_amount_cents, tw_error_count FROM stats_totals WHERE id = 1`,
	).Scan(&twCount, &twCents, &twErrors)
	if err != nil {
		return nil, fmt.Errorf("load stats totals: %w", err)
	}
	return &model.Stats{
		Count:  twCount,
		Amount: float64(twCents) / 100,
		Errors: twErrors,
		TrueMoney: model.StatsPart{
			Amount: float64(twCents) / 100,
			Count:  twCount,
			Errors: twErrors,
		},
	}, nil
}
