package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/internal/validator"
)

type mockTwService struct {
	redeemFn func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error)
}

func (m *mockTwService) Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
	return m.redeemFn(ctx, req)
}

func newTwHandler(fn func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error)) *TwHandler {
	return NewTwHandler(&mockTwService{redeemFn: fn})
}

func TestTwHandler_RedeemPost(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"phone":"0812345678","gift":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed json",
			body:       `{"phone":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown field",
			body:       `{"phone":"0812345678","gift":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","extra":1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "trailing garbage",
			body:       `{"phone":"0812345678","gift":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} extra`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid phone",
			body:       `{"phone":"123","gift":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "empty body",
			body:       ``,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service internal error",
			body:       `{"phone":"0812345678","gift":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`,
			serviceErr: errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTwHandler(func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
				if tt.serviceErr != nil {
					return nil, tt.serviceErr
				}
				if err := validator.ValidateTwRedeem(req); err != nil {
					return nil, err
				}
				return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`)}, nil
			})

			req := httptest.NewRequest(http.MethodPost, "/tw", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			h.RedeemPost(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d, body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var env struct {
					Data map[string]any `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				status, ok := env.Data["status"].(map[string]any)
				if !ok || status["code"] != "SUCCESS" {
					t.Fatalf("got data %v", env.Data)
				}
			}
		})
	}
}

func TestTwHandler_RedeemPost_ValidationDetails(t *testing.T) {
	h := newTwHandler(func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		return nil, &model.ValidationError{Details: model.FieldErrors{
			{Field: "phone", Message: "must be a Thai phone number of exactly 10 digits"},
			{Field: "gift", Message: "gift code must be 20-60 alphanumeric characters"},
		}}
	})

	req := httptest.NewRequest(http.MethodPost, "/tw", bytes.NewBufferString(`{"phone":"1","gift":"x"}`))
	rec := httptest.NewRecorder()

	h.RedeemPost(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	var env struct {
		Error   string             `json:"error"`
		Details []model.FieldError `json:"details"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Details) != 2 {
		t.Fatalf("got %d details, want 2: %s", len(env.Details), rec.Body.String())
	}
}

func TestTwHandler_RedeemGet(t *testing.T) {
	got := make(chan model.TwRedeemRequest, 1)
	h := newTwHandler(func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		got <- req
		return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`)}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/tw/0812345678/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", nil)
	req = withRouteParams(req, map[string]string{"phone": "0812345678", "gift": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	rec := httptest.NewRecorder()

	h.RedeemGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	received := <-got
	if received.Phone != "0812345678" || received.Gift != "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" {
		t.Fatalf("got request %+v", received)
	}
	if !strings.Contains(rec.Body.String(), "SUCCESS") {
		t.Fatalf("body should contain upstream response: %s", rec.Body.String())
	}
}

func TestTwHandler_RedeemGet_InvalidPhone(t *testing.T) {
	h := newTwHandler(func(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
		if err := validator.ValidateTwRedeem(req); err != nil {
			return nil, err
		}
		return &model.TwRedeemResult{Data: json.RawMessage(`{"status":{"code":"SUCCESS"}}`)}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/tw/123/x", nil)
	req = withRouteParams(req, map[string]string{"phone": "123", "gift": "x"})
	rec := httptest.NewRecorder()

	h.RedeemGet(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "phone") {
		t.Fatalf("body should mention field phone: %s", rec.Body.String())
	}
}
