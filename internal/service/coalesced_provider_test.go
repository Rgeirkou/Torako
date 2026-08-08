package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
)

type countingTwProvider struct {
	calls atomic.Int64
}

func (p *countingTwProvider) Redeem(ctx context.Context, req model.TwRedeemRequest) (*model.TwRedeemResult, error) {
	p.calls.Add(1)
	time.Sleep(50 * time.Millisecond)
	return &model.TwRedeemResult{
		Ref:    "voucher-" + req.Gift,
		Amount: "10.00",
	}, nil
}

func TestCoalescedTwProvider_ConcurrentSameKey(t *testing.T) {
	inner := &countingTwProvider{}
	p := NewCoalescedTwProvider(inner)

	req := model.TwRedeemRequest{Phone: "0994322441", Gift: "gift-link"}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := p.Redeem(context.Background(), req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if res.Ref != "voucher-gift-link" {
				t.Errorf("got ref %q", res.Ref)
			}
		}()
	}
	wg.Wait()

	if got := inner.calls.Load(); got != 1 {
		t.Fatalf("inner called %d times, want 1 (coalesced)", got)
	}
}

func TestCoalescedTwProvider_SequentialHitsUpstreamAgain(t *testing.T) {
	inner := &countingTwProvider{}
	p := NewCoalescedTwProvider(inner)

	req := model.TwRedeemRequest{Phone: "0994322441", Gift: "gift-link"}

	for i := 0; i < 3; i++ {
		if _, err := p.Redeem(context.Background(), req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if got := inner.calls.Load(); got != 3 {
		t.Fatalf("inner called %d times, want 3 (no replay from cache)", got)
	}
}
