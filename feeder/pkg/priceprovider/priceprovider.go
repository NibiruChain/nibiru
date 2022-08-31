package priceprovider

type Price struct {
	Symbol string
	Price  float64
}

type Interface interface {
	GetPrices(symbols []string) []Price
	Close()
}
