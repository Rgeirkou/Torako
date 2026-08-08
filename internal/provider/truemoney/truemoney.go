package truemoney

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/internal/validator"
)

const (
	defaultTimeout = 30 * time.Second
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

type Client struct {
	baseURL string
	httpc   *http.Client
	logger  *slog.Logger
}

func NewClient(baseURL string, timeout time.Duration, logger *slog.Logger) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: timeout},
		logger:  logger,
	}
}

type tmRequest struct {
	Mobile      string `json:"mobile"`
	VoucherHash string `json:"voucher_hash"`
}

type tmResponse struct {
	Status struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Data struct {
		Voucher struct {
			VoucherID  string `json:"voucher_id"`
			AmountBaht string `json:"amount_baht"`
			FullName   string `json:"full_name"`
		} `json:"voucher"`
		RedeemerProfile struct {
			MobileNumber string `json:"mobile_number"`
		} `json:"redeemer_profile"`
	} `json:"data"`
}

func (c *Client) Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
	hash, err := extractVoucherHash(req.Gift)
	if err != nil {
		return nil, fmt.Errorf("extract voucher hash: %w", errors.Join(err, model.ErrInvalidInput))
	}

	endpoint := c.baseURL + "/campaign/vouchers/" + url.PathEscape(hash) + "/redeem"
	body, err := json.Marshal(tmRequest{Mobile: req.Phone, VoucherHash: hash})
	if err != nil {
		return nil, fmt.Errorf("encode redeem request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create redeem request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)

	resp, err := c.httpc.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("call truemoney redeem: %w", err)
		}
		return nil, fmt.Errorf("call truemoney redeem: %w", errors.Join(err, model.ErrBadGateway))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read truemoney response: %w", err)
	}

	var parsed tmResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil || parsed.Status.Code == "" {
		c.logger.Error("invalid truemoney response", "status", resp.StatusCode, "err", err)
		return nil, fmt.Errorf("parse truemoney response: %w", model.ErrBadGateway)
	}

	if parsed.Status.Code != "SUCCESS" {
		return nil, c.mapBusinessError(parsed.Status.Code)
	}

	return &model.TwRedeemResult{Data: json.RawMessage(respBody), Amount: parsed.Data.Voucher.AmountBaht, Ref: parsed.Data.Voucher.VoucherID}, nil
}

func (c *Client) mapBusinessError(code string) error {
	c.logger.Warn("truemoney redeem failed", "code", code)
	switch code {
	case "VOUCHER_NOT_FOUND", "INVALID_LINK", "TARGET_NOT_FOUND":
		return fmt.Errorf("truemoney redeem (%s): %w", code, model.ErrNotFound)
	case "VOUCHER_OUT_OF_STOCK":
		return fmt.Errorf("truemoney redeem (%s): %w", code, model.ErrConflict)
	default:
		return &model.ValidationError{Details: model.FieldErrors{
			{Field: "gift", Message: code},
		}}
	}
}

func extractVoucherHash(gift string) (string, error) {
	if !strings.HasPrefix(gift, "http://") && !strings.HasPrefix(gift, "https://") {
		return gift, nil
	}
	u, err := url.Parse(gift)
	if err != nil {
		return "", fmt.Errorf("invalid gift link")
	}
	if !validator.IsTrustedGiftHost(u.Host) {
		return "", fmt.Errorf("gift link host is not trusted")
	}
	hash := u.Query().Get("v")
	if hash == "" {
		return "", fmt.Errorf("gift link does not contain a voucher hash")
	}
	return hash, nil
}
