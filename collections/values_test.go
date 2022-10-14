package collections_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/x/common"
)

func TestProtoValueEncoder(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		protoType, err := common.NewAssetPair("btc:usd")
		require.NoError(t, err)

		registry := testdata.NewTestInterfaceRegistry()
		cdc := codec.NewProtoCodec(registry)

		assertValueBijective[common.AssetPair](t, collections.ProtoValueEncoder[common.AssetPair](cdc), protoType)
	})
}
