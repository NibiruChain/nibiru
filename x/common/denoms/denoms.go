package denoms

// Post Nibiru v2.6.0
const (
	// NOTE: US dollars. Use `denoms.USD` instead of `denoms.UUSD` going forward.
	USD = "usd"
	// Avalon Finance overcollateralized stablecoin.
	// https://github.com/NibiruChain/pricefeeder/pull/64
	USDA  = "usda"
	SUSDA = "susda"
)

// Legacy denoms - These each include an unnecessary "u" prefix for micro.

const ( // stablecoins
	USDC = "uusdc"
	NUSD = "unusd"
	UUSD = "uusd"
	USDT = "uusdt"
)

const ( // volatile assets
	NIBI = "unibi"
	BTC  = "ubtc"
	ETH  = "ueth"
	ATOM = "uatom"
	OSMO = "uosmo"
	AVAX = "uavax"
	SOL  = "usol"
	BNB  = "ubnb"
	ADA  = "uada"
)
