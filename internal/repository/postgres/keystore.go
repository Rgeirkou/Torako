package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgeirkou/tyrako/internal/model"
)

const keyColumns = `id, name, rank, key_hash, scopes, expires_at, revoked_at, last_used_at, request_count, created_at`

type KeyStore struct {
	pool *pgxpool.Pool
}

func NewKeyStore(pool *pgxpool.Pool) *KeyStore {
	return &KeyStore{pool: pool}
}

func (s *KeyStore) Create(ctx context.Context, key *model.ApiKey) error {
	scopes := key.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	rank := key.Rank
	if rank == "" {
		rank = "member"
	}
	row := s.pool.QueryRow(ctx,
		`INSERT INTO api_keys (name, rank, key_hash, scopes, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		key.Name, rank, key.KeyHash, scopes, key.ExpiresAt)
	if err := row.Scan(&key.ID, &key.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return model.ErrConflict
		}
		return fmt.Errorf("insert api key: %w", err)
	}
	key.Rank = rank
	return nil
}

func (s *KeyStore) GetByHash(ctx context.Context, hash string) (*model.ApiKey, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+keyColumns+` FROM api_keys WHERE key_hash = $1`, hash)
	key, err := scanKey(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return key, nil
}

func (s *KeyStore) List(ctx context.Context) ([]model.ApiKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+keyColumns+` FROM api_keys ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	keys := make([]model.ApiKey, 0)
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, *key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return keys, nil
}

func (s *KeyStore) Revoke(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (s *KeyStore) Touch(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = now(), request_count = request_count + 1 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("touch api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func scanKey(row pgx.Row) (*model.ApiKey, error) {
	var key model.ApiKey
	err := row.Scan(&key.ID, &key.Name, &key.Rank, &key.KeyHash, &key.Scopes,
		&key.ExpiresAt, &key.RevokedAt, &key.LastUsedAt, &key.RequestCount, &key.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
