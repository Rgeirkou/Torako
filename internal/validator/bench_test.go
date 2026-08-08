package validator

import (
	"strings"
	"testing"

	"github.com/rgeirkou/tyrako/internal/model"
)

func BenchmarkValidateTwRedeem_Code(b *testing.B) {
	req := model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ValidateTwRedeem(req)
	}
}

func BenchmarkValidateTwRedeem_Link(b *testing.B) {
	req := model.TwRedeemRequest{Phone: "0812345678", Gift: "https://gift.truemoney.com/campaign/voucher_detail?v=abcdefghijklmnopqrstuvwxyz1234567890"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ValidateTwRedeem(req)
	}
}
