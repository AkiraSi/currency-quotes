package rate

import (
	"time"

	"currency-quotes/common/currencies"
)

type Rate struct {
	Code      currencies.Currency
	Nominal   int64
	Value     int64
	UpdatedAt time.Time
}
