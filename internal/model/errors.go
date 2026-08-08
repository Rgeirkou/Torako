package model

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrRateLimited  = errors.New("rate limit exceeded")
	ErrBadRequest   = errors.New("bad request")
	ErrBadGateway   = errors.New("upstream service unavailable")
)
