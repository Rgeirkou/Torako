package handler

import (
	"context"
	"net/http"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/pkg/response"
)

type twService interface {
	Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error)
}

type TwHandler struct {
	svc twService
}

func NewTwHandler(svc twService) *TwHandler {
	return &TwHandler{svc: svc}
}

func (h *TwHandler) RedeemPost(w http.ResponseWriter, r *http.Request) {
	var req model.TwRedeemRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.svc.Redeem(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res.Data)
}

func (h *TwHandler) RedeemGet(w http.ResponseWriter, r *http.Request) {
	phone, err := pathParam(r, "phone")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid phone parameter")
		return
	}
	gift, err := pathParam(r, "gift")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid gift parameter")
		return
	}
	req := model.TwRedeemRequest{Phone: phone, Gift: gift}
	res, err := h.svc.Redeem(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res.Data)
}
