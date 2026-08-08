package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rgeirkou/tyrako/internal/model"
)

type mockStatsStore struct {
	totalsFn func(ctx context.Context) (*model.Stats, error)
}

func (m *mockStatsStore) Totals(ctx context.Context) (*model.Stats, error) {
	return m.totalsFn(ctx)
}

func TestStatsHandler_Stats(t *testing.T) {
	handler := NewStatsHandler(&mockStatsStore{totalsFn: func(ctx context.Context) (*model.Stats, error) {
		return &model.Stats{
			Amount: 102.50,
			Count:  2,
			TrueMoney: model.StatsPart{
				Amount: 102.50,
				Count:  2,
			},
		}, nil
	}})

	rec := httptest.NewRecorder()
	handler.Stats(rec, httptest.NewRequest(http.MethodGet, "/stats", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	var body map[string]model.Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	stats, ok := body["data"]
	if !ok {
		t.Fatalf("response must use the data envelope, got %s", rec.Body.String())
	}
	if stats.Amount != 102.50 || stats.Count != 2 {
		t.Fatalf("totals = %+v, want amount 102.50 count 2", stats)
	}
	if stats.TrueMoney.Amount != 102.50 || stats.TrueMoney.Count != 2 {
		t.Fatalf("truemoney = %+v", stats.TrueMoney)
	}
}
