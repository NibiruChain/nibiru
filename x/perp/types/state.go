package types

import "github.com/NibiruChain/nibiru/x/common"

func (p *Position) GetAssetPair() common.AssetPair {
	pair, err := common.NewAssetPairFromStr(p.Pair)
	if err != nil {
		panic(err)
	}

	return pair
}
