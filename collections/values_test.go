package collections

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/gogo/protobuf/types"
)

func TestProtoValueEncoder(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		protoType := types.BytesValue{Value: []byte("testing")}

		registry := testdata.NewTestInterfaceRegistry()
		cdc := codec.NewProtoCodec(registry)

		assertValueBijective[types.BytesValue](t, ProtoValueEncoder[types.BytesValue](cdc), protoType)
	})
}

func TestDecValueEncoder(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		assertValueBijective(t, DecValueEncoder, sdk.MustNewDecFromStr("-1000.5858"))
	})
}

func TestAccAddressValueEncoder(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		assertValueBijective(t, AccAddressValueEncoder, testutil.AccAddress())
	})
}

func TestUint64ValueEncoder(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		assertValueBijective(t, Uint64ValueEncoder, 1000)
	})
}
