package cbr

import (
	"currency-quotes/common/currencies"
	"encoding/json"
	"time"
)

type Response struct {
	Date   time.Time           `json:"Timestamp"`
	Valute map[string]Currency `json:"Valute"`
}

type Currency struct {
	NumCode int64 `json:"NumCode"`
	Nominal int64 `json:"Nominal"`
	Value   int64 `json:"Value"`
}

type aliesResponse struct {
	Timestamp time.Time `json:"Timestamp"`
	Valute    map[string]struct {
		NumCode string  `json:"NumCode"`
		Nominal int64   `json:"Nominal"`
		Value   float64 `json:"Value"`
	} `json:"Valute"`
}

func (r *Response) unmarshalJSON(b []byte) error {
	var alias aliesResponse
	if err := json.Unmarshal(b, &alias); err != nil {
		return err
	}

	r.Date = alias.Timestamp
	r.Valute = make(map[string]Currency, len(alias.Valute))

	for key, v := range alias.Valute {
		digitCode := currencies.CodeByCurrency(key)
		if digitCode == currencies.CodeErr {
			continue
		}

		r.Valute[key] = Currency{
			NumCode: int64(digitCode),
			Nominal: v.Nominal,
			Value:   int64(v.Value * 10000),
		}
	}

	return nil
}
