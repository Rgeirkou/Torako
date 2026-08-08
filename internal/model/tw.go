package model

import "encoding/json"

type TwRedeemRequest struct {
	Phone string `json:"phone"`
	Gift  string `json:"gift"`
}

type TwRedeemResult struct {
	Data json.RawMessage
	// Amount is the voucher amount_baht string from the upstream response,
	// used for statistics. It is empty when the upstream omits it.
	Amount string
	// Ref is the voucher id from the upstream response, used to avoid
	// double-counting repeated redemptions of the same voucher.
	Ref string
}
