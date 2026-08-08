package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rgeirkou/tyrako/internal/model"
)

type mockTwProvider struct {
	redeemFn func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error)
}

func (m *mockTwProvider) Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
	return m.redeemFn(ctx, req)
}

func TestTwService_Redeem(t *testing.T) {
	valid := model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}

	tests := []struct {
		name     string
		req      model.TwRedeemRequest
		provider func() *mockTwProvider
		wantErr  bool
	}{
		{
			name: "success",
			req:  valid,
			provider: func() *mockTwProvider {
				return &mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
					return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`)}, nil
				}}
			},
		},
		{
			name: "invalid input rejected before provider",
			req:  model.TwRedeemRequest{Phone: "123", Gift: "x"},
			provider: func() *mockTwProvider {
				return &mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
					t.Fatal("provider must not be called for invalid input")
					return nil, nil
				}}
			},
			wantErr: true,
		},
		{
			name: "provider error propagates",
			req:  valid,
			provider: func() *mockTwProvider {
				return &mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
					return nil, errors.New("upstream down")
				}}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTwService(tt.provider(), nil, nil)

			res, err := svc.Redeem(context.Background(), tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Contains(res.Data, []byte(`"code":"SUCCESS"`)) {
				t.Fatalf("got data %s, want upstream response", res.Data)
			}
		})
	}
}

func TestTwService_Redeem_ReturnsValidationError(t *testing.T) {
	svc := NewTwService(&mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		t.Fatal("provider must not be called for invalid input")
		return nil, nil
	}}, nil, nil)

	_, err := svc.Redeem(context.Background(), model.TwRedeemRequest{Phone: "123", Gift: "x"})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

type fakeStats struct {
	twCount, twCents, twErrors int64
	err                        error
}

func (f *fakeStats) RecordTw(_ context.Context, amountCents int64, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.twCount++
	f.twCents += amountCents
	return nil
}

func (f *fakeStats) RecordTwError(_ context.Context) error {
	if f.err != nil {
		return f.err
	}
	f.twErrors++
	return nil
}

func TestTwService_Redeem_RecordsStatsOnSuccess(t *testing.T) {
	stats := &fakeStats{}
	svc := NewTwService(&mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`), Amount: "100.00"}, nil
	}}, stats, nil)

	if _, err := svc.Redeem(context.Background(), model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.twCount != 1 || stats.twCents != 10000 {
		t.Fatalf("stats = count %d cents %d, want 1 and 10000", stats.twCount, stats.twCents)
	}
}

func TestTwService_Redeem_UnparseableAmountStillCounts(t *testing.T) {
	stats := &fakeStats{}
	svc := NewTwService(&mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`), Amount: "unknown"}, nil
	}}, stats, nil)

	if _, err := svc.Redeem(context.Background(), model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.twCount != 1 || stats.twCents != 0 {
		t.Fatalf("stats = count %d cents %d, want 1 and 0 (amount must not fail the redeem)", stats.twCount, stats.twCents)
	}
}

func TestTwService_Redeem_FailureRecordedAsError(t *testing.T) {
	stats := &fakeStats{}
	svc := NewTwService(&mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		return nil, errors.New("upstream down")
	}}, stats, nil)

	if _, err := svc.Redeem(context.Background(), model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}); err == nil {
		t.Fatal("want error from upstream")
	}
	if stats.twCount != 0 {
		t.Fatalf("failed redeem must not add to success count, got count %d", stats.twCount)
	}
	if stats.twErrors != 1 {
		t.Fatalf("failed redeem must be counted as an error, got %d", stats.twErrors)
	}
}

func TestTwService_Redeem_ValidationErrorRecorded(t *testing.T) {
	stats := &fakeStats{}
	svc := NewTwService(&mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		t.Fatal("provider must not be called for an invalid request")
		return nil, nil
	}}, stats, nil)

	if _, err := svc.Redeem(context.Background(), model.TwRedeemRequest{Phone: "", Gift: ""}); err == nil {
		t.Fatal("want validation error")
	}
	if stats.twErrors != 1 {
		t.Fatalf("invalid request must be counted as an error, got %d", stats.twErrors)
	}
}

func TestTwService_Redeem_StatsErrorDoesNotFailRedeem(t *testing.T) {
	stats := &fakeStats{err: errors.New("db down")}
	svc := NewTwService(&mockTwProvider{redeemFn: func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`), Amount: "100.00"}, nil
	}}, stats, nil)

	if _, err := svc.Redeem(context.Background(), model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}); err != nil {
		t.Fatalf("stats failure must not fail the redeem, got %v", err)
	}
}
