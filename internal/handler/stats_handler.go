package handler

import (
	"context"
	"net/http"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/pkg/response"
)

type statsStore interface {
	Totals(ctx context.Context) (*model.Stats, error)
}

type StatsHandler struct {
	store statsStore
}

func NewStatsHandler(store statsStore) *StatsHandler {
	return &StatsHandler{store: store}
}

func (h *StatsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.Totals(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}
