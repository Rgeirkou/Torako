package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/internal/money"
	"github.com/rgeirkou/tyrako/internal/validator"
)

type twProvider interface {
	Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error)
}

type TwService struct {
	provider twProvider
	stats    statsStore
	logger   *slog.Logger
}

// NewTwService builds the TrueMoney redeem service. stats and logger may be
// nil; stats recording is best-effort and never fails the operation.
func NewTwService(provider twProvider, stats statsStore, logger *slog.Logger) *TwService {
	return &TwService{provider: provider, stats: stats, logger: logger}
}

func (s *TwService) Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
	if err := validator.ValidateTwRedeem(req); err != nil {
		s.recordTwError()
		return nil, fmt.Errorf("validate redeem request: %w", err)
	}
	res, err := s.provider.Redeem(ctx, req)
	if err != nil {
		s.recordTwError()
		return nil, fmt.Errorf("redeem gift: %w", err)
	}
	s.record(res)
	return res, nil
}

// recordTwError counts a failed redemption attempt.
func (s *TwService) recordTwError() {
	if s.stats == nil {
		return
	}
	if err := s.stats.RecordTwError(context.Background()); err != nil {
		if s.logger != nil {
			s.logger.Warn("record truemoney error stat", "err", err)
		}
	}
}

// record counts a successful redeem. A voucher amount that cannot be parsed
// still counts, with a zero amount and a warning.
func (s *TwService) record(res *model.TwRedeemResult) {
	if s.stats == nil {
		return
	}
	var cents int64
	if res.Amount != "" {
		parsed, err := money.ParseCents(res.Amount)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("parse truemoney voucher amount", "amount", res.Amount, "err", err)
			}
		} else {
			cents = parsed
		}
	}
	if err := s.stats.RecordTw(context.Background(), cents, res.Ref); err != nil {
		if s.logger != nil {
			s.logger.Warn("record truemoney stat", "err", err)
		}
	}
}
