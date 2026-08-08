package service

import (
	"context"
)

// statsStore records successful and failed operations for the /stats
// endpoint. It is optional: services degrade gracefully (no recording)
// when it is nil.
//
// ref identifies the unique upstream operation (voucher id).
// Stores deduplicate successful records on it, so repeating the same
// operation never counts twice. An empty ref is always counted. Error
// records are never deduplicated: each failed attempt is counted.
type statsStore interface {
	RecordTw(ctx context.Context, amountCents int64, ref string) error
	RecordTwError(ctx context.Context) error
}
