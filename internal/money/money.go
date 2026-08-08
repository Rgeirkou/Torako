// Package money converts monetary strings and floats to integer cents so
// statistics can be aggregated exactly without floating point drift.
package money

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var amountRe = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,2})?$`)

// ParseCents converts a decimal amount string such as "100.00" or "100.5"
// to integer cents (10000 and 10050). At most two decimal places are
// accepted; anything else is rejected so no rounding decision is implicit.
func ParseCents(s string) (int64, error) {
	if !amountRe.MatchString(s) {
		return 0, fmt.Errorf("invalid amount %q", s)
	}
	whole, frac, _ := strings.Cut(s, ".")
	if frac == "" {
		frac = "00"
	}
	if len(frac) == 1 {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	if w > (math.MaxInt64-f)/100 {
		return 0, fmt.Errorf("amount %q overflows", s)
	}
	return w*100 + f, nil
}

// FloatCents converts a float amount to integer cents, clamped at zero.
func FloatCents(f float64) int64 {
	if f <= 0 {
		return 0
	}
	return int64(math.Round(f * 100))
}
