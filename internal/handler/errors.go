package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/pkg/response"
)

func writeServiceError(w http.ResponseWriter, err error) {
	var ve *model.ValidationError
	switch {
	case errors.As(err, &ve):
		response.ErrorDetails(w, http.StatusUnprocessableEntity, "invalid input", ve.Details)
	case errors.Is(err, model.ErrNotFound):
		response.Error(w, http.StatusNotFound, "not found")
	case errors.Is(err, model.ErrConflict):
		response.Error(w, http.StatusConflict, "conflict")
	case errors.Is(err, model.ErrUnauthorized):
		response.Error(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, model.ErrForbidden):
		response.Error(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, model.ErrRateLimited):
		response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
	case errors.Is(err, model.ErrBadRequest):
		response.Error(w, http.StatusBadRequest, "bad request")
	case errors.Is(err, model.ErrBadGateway):
		response.Error(w, http.StatusBadGateway, "upstream service unavailable")
	case errors.Is(err, context.DeadlineExceeded):
		response.Error(w, http.StatusGatewayTimeout, "upstream timeout")
	case errors.Is(err, model.ErrInvalidInput):
		response.Error(w, http.StatusUnprocessableEntity, "invalid input")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
