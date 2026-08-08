package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/pkg/response"
)

type keyService interface {
	CreateKey(ctx context.Context, in model.CreateApiKeyInput) (model.ApiKeyView, error)
	ListKeys(ctx context.Context) ([]model.ApiKeyView, error)
	RevokeKey(ctx context.Context, id int64) error
	RotateKey(ctx context.Context, id int64) (model.ApiKeyView, error)
}

type KeyHandler struct {
	svc keyService
}

func NewKeyHandler(svc keyService) *KeyHandler {
	return &KeyHandler{svc: svc}
}

func (h *KeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in model.CreateApiKeyInput
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	view, err := h.svc.CreateKey(r.Context(), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, view)
}

func (h *KeyHandler) List(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.ListKeys(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, views)
}

func (h *KeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id parameter")
		return
	}
	if err := h.svc.RevokeKey(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *KeyHandler) Rotate(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id parameter")
		return
	}
	view, err := h.svc.RotateKey(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, view)
}

func idParam(r *http.Request) (int64, error) {
	raw, err := pathParam(r, "id")
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id parameter")
	}
	return id, nil
}
