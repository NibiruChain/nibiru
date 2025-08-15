package asset

import (
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
)

const (
	PAIR_BTC  Pair = denoms.BTC + ":" + denoms.UUSD
	PAIR_ETH  Pair = denoms.ETH + ":" + denoms.UUSD
	PAIR_ATOM Pair = denoms.ATOM + ":" + denoms.UUSD
	PAIR_USDC Pair = denoms.USDC + ":" + denoms.UUSD
	PAIR_USDT Pair = denoms.USDT + ":" + denoms.UUSD
)
