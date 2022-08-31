package oracle

var _ PrevotesCache = (*MemPrevoteCache)(nil)

type MemPrevoteCache struct {
	salt, exchangeRatesStr, feeder string
}

func (m *MemPrevoteCache) SetPrevote(salt string, exchangeRatesStr, feeder string) {
	m.salt, m.exchangeRatesStr, m.feeder = salt, exchangeRatesStr, feeder
}

func (m *MemPrevoteCache) GetPrevote() (salt, exchangeRatesStr, feeder string, ok bool) {
	if m.salt == "" {
		return "", "", "", false
	}

	return m.salt, m.exchangeRatesStr, m.feeder, true
}
