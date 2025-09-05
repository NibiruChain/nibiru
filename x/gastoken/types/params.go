package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func (p Params) Validate() error {
	if !common.IsHexAddress(p.UniswapV3SwapRouterAddress) {
		return fmt.Errorf("invalid uniswap_v3_swap_router_address: %q", p.UniswapV3SwapRouterAddress)
	}
	if !common.IsHexAddress(p.UniswapV3QuoterAddress) {
		return fmt.Errorf("invalid uniswap_v3_quoter_address: %q", p.UniswapV3QuoterAddress)
	}
	if !common.IsHexAddress(p.WnibiAddress) {
		return fmt.Errorf("invalid wnibi_address: %q", p.WnibiAddress)
	}
	return nil
}
