package model

// Stats is the all-time aggregate served by GET /stats. Amounts are
// decimal baht; internal storage is integer cents. Errors counts every
// failed operation attempt (validation or upstream failure).
type Stats struct {
	Amount    float64   `json:"amount"`
	Count     int64     `json:"count"`
	Errors    int64     `json:"errors"`
	TrueMoney StatsPart `json:"truemoney"`
}

type StatsPart struct {
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
	Errors int64   `json:"errors"`
}
