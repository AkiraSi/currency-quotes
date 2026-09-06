package internal

import (
	"currency-quotes/common/currencies"
	"currency-quotes/common/rate"
	"sync"
)

type Updater struct {
	mu    sync.RWMutex
	rates map[currencies.Currency]rate.Rate
}

func NewUpdater() *Updater {
	return &Updater{
		rates: getDefaultCurrencies(),
		mu:    sync.RWMutex{},
	}
}

func getDefaultCurrencies() map[currencies.Currency]rate.Rate {
	return map[currencies.Currency]rate.Rate{
		currencies.CodeRub: {
			Code:    currencies.CodeRub,
			Nominal: 1,
			Value:   1_00,
		},
		currencies.CodeEur: {
			Code:    currencies.CodeEur,
			Nominal: 1,
			Value:   10056_93,
		},
		currencies.CodeUSD: {
			Code:    currencies.CodeUSD,
			Nominal: 1,
			Value:   8658_57,
		},
	}
}
