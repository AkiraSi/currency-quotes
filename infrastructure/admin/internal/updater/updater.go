package updater

import (
	"currency-quotes/common/currencies"
	"currency-quotes/common/rate"
	"currency-quotes/pkg/clients/cbr"
	"sync"
	"time"
)

type Updater struct {
	mu    sync.RWMutex
	Rates map[currencies.Currency]rate.Rate

	updateChannel chan bool

	cbrClient *cbr.Client
}

func NewUpdater() *Updater {
	return &Updater{
		Rates:     getDefaultCurrencies(),
		cbrClient: cbr.NewClient(),
	}
}

func (u *Updater) UpdateCurrencies() error {
	actualCurrencies, err := u.cbrClient.GetCurrencies()
	if err != nil {
		return err
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	for _, actualCurrency := range actualCurrencies.Valute {
		code := currencies.Currency(actualCurrency.NumCode)

		u.Rates[code] = rate.Rate{
			Code:      code,
			Nominal:   actualCurrency.Nominal,
			Value:     actualCurrency.Value,
			UpdatedAt: actualCurrencies.Date,
		}
	}

	return nil
}

func getDefaultCurrencies() map[currencies.Currency]rate.Rate {
	now := time.Now()

	return map[currencies.Currency]rate.Rate{
		currencies.CodeRub: {
			Code:      currencies.CodeRub,
			Nominal:   1,
			Value:     1_00,
			UpdatedAt: now,
		},
		currencies.CodeEur: {
			Code:      currencies.CodeEur,
			Nominal:   1,
			Value:     10056_93,
			UpdatedAt: now,
		},
		currencies.CodeUSD: {
			Code:      currencies.CodeUSD,
			Nominal:   1,
			Value:     8658_57,
			UpdatedAt: now,
		},
	}
}
