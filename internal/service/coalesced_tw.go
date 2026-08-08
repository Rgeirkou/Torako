package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/rgeirkou/tyrako/internal/model"
)

// CoalescedTwProvider deduplicates concurrent redemptions of the same
// (phone, gift) pair so a retry storm triggers a single upstream call.
// Nothing is stored after the call completes: replaying the same gift
// later goes straight to TrueMoney, which answers with the real state
// (e.g. VOUCHER_NOT_FOUND after the voucher was spent), so a consumed
// voucher can never be served as a fake success from a cache.
type CoalescedTwProvider struct {
	inner twProvider
	group singleflight.Group
}

func NewCoalescedTwProvider(inner twProvider) *CoalescedTwProvider {
	return &CoalescedTwProvider{inner: inner}
}

func (p *CoalescedTwProvider) Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 35*time.Second)
	defer cancel()

	v, err, _ := p.group.Do(twCoalesceKey(req.Phone, req.Gift), func() (any, error) {
		return p.inner.Redeem(callCtx, req)
	})
	if err != nil {
		return nil, err
	}
	res, ok := v.(*model.TwRedeemResult)
	if !ok {
		return nil, fmt.Errorf("coalesced tw provider: unexpected result type %T", v)
	}
	return res, nil
}

func twCoalesceKey(phone, gift string) string {
	sum := sha256.Sum256([]byte("tw|" + phone + "|" + gift))
	return hex.EncodeToString(sum[:])
}
