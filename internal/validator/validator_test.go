package validator

import (
	"errors"
	"strings"
	"testing"

	"github.com/rgeirkou/tyrako/internal/model"
)

func TestValidateThaiPhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{name: "valid", phone: "0812345678"},
		{name: "valid nine prefix", phone: "0999999999"},
		{name: "empty", phone: "", wantErr: true},
		{name: "not starting with 0", phone: "1812345678", wantErr: true},
		{name: "too short", phone: "081234567", wantErr: true},
		{name: "too long", phone: "08123456789", wantErr: true},
		{name: "contains letters", phone: "081234567a", wantErr: true},
		{name: "contains dash", phone: "081-234-567", wantErr: true},
		{name: "spaces", phone: "081 234 5678", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateThaiPhone(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateThaiPhone(%q) err=%v, wantErr=%v", tt.phone, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGift(t *testing.T) {
	validCode := strings.Repeat("A", 30)
	longLink := "https://gift.truemoney.com/campaign/?v=" + strings.Repeat("x", 3000)

	tests := []struct {
		name    string
		gift    string
		wantErr bool
	}{
		{name: "valid code", gift: validCode},
		{name: "valid code lower", gift: strings.Repeat("a", 20)},
		{name: "valid code with digits", gift: strings.Repeat("x", 35)},
		{name: "valid link", gift: "https://gift.truemoney.com/campaign/?v=abc123"},
		{name: "valid link www", gift: "http://gift.truemoney.com/x"},
		{name: "empty", gift: "", wantErr: true},
		{name: "code too short", gift: strings.Repeat("A", 19), wantErr: true},
		{name: "code too long", gift: strings.Repeat("A", 61), wantErr: true},
		{name: "code with dash", gift: strings.Repeat("A", 20) + "-", wantErr: true},
		{name: "link untrusted host", gift: "https://evil.com/x", wantErr: true},
		{name: "link spoofed suffix", gift: "https://evil-truemoney.com/x", wantErr: true},
		{name: "link spoofed suffix 2", gift: "https://truemoney.com.evil.com/x", wantErr: true},
		{name: "link no host", gift: "https:///x", wantErr: true},
		{name: "link userinfo before trusted host", gift: "https://evil.com@truemoney.com/x", wantErr: false},
		{name: "link userinfo after trusted host", gift: "https://truemoney.com@evil.com/x", wantErr: true},
		{name: "link double userinfo", gift: "https://truemoney.com@evil.com@evil2.com/x", wantErr: true},
		{name: "link trailing dot host", gift: "https://truemoney.com./x", wantErr: true},
		{name: "link percent-encoded host", gift: "https://truemoney%2Ecom/x", wantErr: true},
		{name: "link uppercase host", gift: "https://TRUEMONEY.COM/x", wantErr: true},
		{name: "link explicit port", gift: "https://truemoney.com:443/x", wantErr: true},
		{name: "link punycode subdomain", gift: "https://xn--bcher-kva.truemoney.com/x", wantErr: false},
		{name: "link fragment contains trusted", gift: "https://evil.com/#truemoney.com", wantErr: true},
		{name: "link too long", gift: longLink, wantErr: true},
		{name: "garbage", gift: "http://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGift(tt.gift)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateGift(%q) err=%v, wantErr=%v", tt.gift, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTwRedeem(t *testing.T) {
	tests := []struct {
		name       string
		req        model.TwRedeemRequest
		wantFields int
	}{
		{name: "valid", req: model.TwRedeemRequest{Phone: "0812345678", Gift: strings.Repeat("A", 30)}},
		{name: "both invalid", req: model.TwRedeemRequest{Phone: "123", Gift: "x"}, wantFields: 2},
		{name: "phone invalid only", req: model.TwRedeemRequest{Phone: "081", Gift: strings.Repeat("A", 30)}, wantFields: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTwRedeem(tt.req)
			if tt.wantFields == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *model.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *model.ValidationError, got %T: %v", err, err)
			}
			if len(ve.Details) != tt.wantFields {
				t.Fatalf("got %d field errors, want %d: %+v", len(ve.Details), tt.wantFields, ve.Details)
			}
		})
	}
}
