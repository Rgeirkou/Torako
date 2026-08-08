package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/rgeirkou/tyrako/internal/model"
)

var (
	phoneRe = regexp.MustCompile(`^0[0-9]{9}$`)
	giftRe  = regexp.MustCompile(`^[A-Za-z0-9]{20,60}$`)
)

const (
	maxGiftLinkLen = 2048
)

func ValidateThaiPhone(v string) error {
	if !phoneRe.MatchString(v) {
		return fmt.Errorf("must be a Thai phone number of exactly 10 digits")
	}
	return nil
}

// IsTrustedGiftHost reports whether host is truemoney.com or a subdomain of
// it. It is shared by the validator and the TrueMoney provider so the
// security check cannot drift between the two.
func IsTrustedGiftHost(host string) bool {
	return host == "truemoney.com" || strings.HasSuffix(host, ".truemoney.com")
}

func ValidateGift(v string) error {
	if strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") {
		if len(v) > maxGiftLinkLen {
			return fmt.Errorf("gift link is too long")
		}
		u, err := url.Parse(v)
		if err != nil || u.Host == "" {
			return fmt.Errorf("invalid gift link")
		}
		if !IsTrustedGiftHost(u.Host) {
			return fmt.Errorf("gift link host is not trusted")
		}
		return nil
	}
	if !giftRe.MatchString(v) {
		return fmt.Errorf("gift code must be 20-60 alphanumeric characters")
	}
	return nil
}

func ValidateTwRedeem(req model.TwRedeemRequest) error {
	var errs model.FieldErrors
	errs.Add("phone", ValidateThaiPhone(req.Phone))
	errs.Add("gift", ValidateGift(req.Gift))
	if len(errs) > 0 {
		return &model.ValidationError{Details: errs}
	}
	return nil
}
