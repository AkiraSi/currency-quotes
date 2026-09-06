package currencies

type Currency uint16

const (
	RUB = "RUB"
	EUR = "EUR"
	USD = "USD"
	ERR = ""

	CodeRub Currency = 643
	CodeEur Currency = 978
	CodeUSD Currency = 840
	CodeErr Currency = 0
)

var currencies = map[Currency]string{
	CodeRub: RUB,
	CodeEur: EUR,
	CodeUSD: USD,
	CodeErr: ERR,
}

func (c Currency) String() string {
	return currencies[c]
}

var currencyToString = map[Currency]string{
	CodeRub: RUB,
	CodeEur: EUR,
	CodeUSD: USD,
}

func CurrencyByCode(code Currency) string {
	return code.String()
}

var stringToCurrency = map[string]Currency{
	RUB: CodeRub,
	EUR: CodeEur,
	USD: CodeUSD,
}

func CodeByCurrency(currency string) Currency {
	if code, ok := stringToCurrency[currency]; ok {
		return code
	}

	return CodeErr
}
