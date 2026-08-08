package truemoney

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const successBody = `{
	"status": {"code": "SUCCESS", "message": "SUCCESS"},
	"data": {
		"voucher": {
			"voucher_id": "V123",
			"amount_baht": "100.00",
			"full_name": "Test User"
		},
		"redeemer_profile": {"mobile_number": "0812345678"}
	}
}`

func writeJSON(w http.ResponseWriter, raw string) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	_ = json.Compact(&buf, []byte(raw))
	_, _ = w.Write(buf.Bytes())
}

func TestClient_Redeem_Success_FromLink(t *testing.T) {
	var gotBody tmRequest
	var gotPath string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		writeJSON(w, successBody)
	}))
	defer upstream.Close()

	client := NewClient(upstream.URL, 30*time.Second, discardLogger())
	res, err := client.Redeem(context.Background(), model.TwRedeemRequest{
		Phone: "0812345678",
		Gift:  "https://gift.truemoney.com/campaign/voucher_detail?v=abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/campaign/vouchers/abc123/redeem" {
		t.Fatalf("got path %q, want /campaign/vouchers/abc123/redeem", gotPath)
	}
	if gotBody.Mobile != "0812345678" || gotBody.VoucherHash != "abc123" {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if !bytes.Contains(res.Data, []byte(`"code":"SUCCESS"`)) {
		t.Fatalf("unexpected data: %s", res.Data)
	}
	if !bytes.Contains(res.Data, []byte(`"amount_baht":"100.00"`)) {
		t.Fatalf("unexpected data: %s", res.Data)
	}
	if res.Amount != "100.00" {
		t.Fatalf("Amount = %q, want %q", res.Amount, "100.00")
	}
}

func TestClient_Redeem_Success_FromCode(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, successBody)
	}))
	defer upstream.Close()

	client := NewClient(upstream.URL, 30*time.Second, discardLogger())
	res, err := client.Redeem(context.Background(), model.TwRedeemRequest{
		Phone: "0812345678",
		Gift:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(res.Data, []byte(`"code":"SUCCESS"`)) {
		t.Fatalf("unexpected data: %s", res.Data)
	}
}

func TestClient_Redeem_BusinessErrors(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		wantErr    error
		wantStatus error
	}{
		{name: "voucher not found", code: "VOUCHER_NOT_FOUND", wantErr: model.ErrNotFound},
		{name: "invalid link", code: "INVALID_LINK", wantErr: model.ErrNotFound},
		{name: "voucher out of stock", code: "VOUCHER_OUT_OF_STOCK", wantErr: model.ErrConflict},
		{name: "voucher expired", code: "VOUCHER_EXPIRED", wantErr: model.ErrInvalidInput},
		{name: "cannot get own voucher", code: "CANNOT_GET_OWN_VOUCHER", wantErr: model.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":{"code":"` + tt.code + `","message":"x"}}`))
			}))
			defer upstream.Close()

			client := NewClient(upstream.URL, 30*time.Second, discardLogger())
			_, err := client.Redeem(context.Background(), model.TwRedeemRequest{
				Phone: "0812345678",
				Gift:  "https://gift.truemoney.com/campaign/?v=abc123",
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Redeem_Expired_HasFieldDetails(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":{"code":"VOUCHER_EXPIRED","message":"x"}}`))
	}))
	defer upstream.Close()

	client := NewClient(upstream.URL, 30*time.Second, discardLogger())
	_, err := client.Redeem(context.Background(), model.TwRedeemRequest{
		Phone: "0812345678",
		Gift:  "https://gift.truemoney.com/campaign/?v=abc123",
	})
	var ve *model.ValidationError
	if !errors.As(err, &ve) || len(ve.Details) != 1 || ve.Details[0].Field != "gift" {
		t.Fatalf("got %v, want validation error on gift", err)
	}
}

func TestClient_Redeem_LinkWithoutHash(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", 1*time.Second, discardLogger())
	_, err := client.Redeem(context.Background(), model.TwRedeemRequest{
		Phone: "0812345678",
		Gift:  "https://gift.truemoney.com/campaign/voucher_detail",
	})
	if err == nil || !errors.Is(err, model.ErrInvalidInput) {
		t.Fatalf("got err %v, want ErrInvalidInput", err)
	}
}

func TestClient_Redeem_NetworkError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", 1*time.Second, discardLogger())
	_, err := client.Redeem(context.Background(), model.TwRedeemRequest{
		Phone: "0812345678",
		Gift:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err == nil || !errors.Is(err, model.ErrBadGateway) {
		t.Fatalf("got err %v, want ErrBadGateway", err)
	}
}

func TestClient_Redeem_InvalidResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>error</html>"))
	}))
	defer upstream.Close()

	client := NewClient(upstream.URL, 30*time.Second, discardLogger())
	_, err := client.Redeem(context.Background(), model.TwRedeemRequest{
		Phone: "0812345678",
		Gift:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if !errors.Is(err, model.ErrBadGateway) {
		t.Fatalf("got err %v, want ErrBadGateway", err)
	}
}

func TestExtractVoucherHash(t *testing.T) {
	tests := []struct {
		gift    string
		want    string
		wantErr bool
	}{
		{gift: "abcdefghijklmnopqrstuvwxyz", want: "abcdefghijklmnopqrstuvwxyz"},
		{gift: "https://gift.truemoney.com/campaign/?v=hash123", want: "hash123"},
		{gift: "https://gift.truemoney.com/campaign/voucher_detail?v=hash456", want: "hash456"},
		{gift: "https://truemoney.com/?v=hash", want: "hash"},
		{gift: "https://gift.truemoney.com/campaign/voucher_detail", wantErr: true},
		{gift: "https://evil.com/?v=hash", wantErr: true},
		{gift: "https://evil-truemoney.com/?v=hash", wantErr: true},
		{gift: "https://truemoney.com.evil.com/?v=hash", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.gift, func(t *testing.T) {
			got, err := extractVoucherHash(tt.gift)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
