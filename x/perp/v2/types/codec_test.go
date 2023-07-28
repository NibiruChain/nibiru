package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()
	RegisterLegacyAminoCodec(cdc)

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	RegisterInterfaces(interfaceRegistry)

	msgs := []sdk.Msg{
		&MsgAddMargin{},
		&MsgRemoveMargin{},
		&MsgMarketOrder{},
		&MsgClosePosition{},
		&MsgPartialClose{},
		&MsgDonateToEcosystemFund{},
		&MsgMultiLiquidate{},
	}

	for _, msg := range msgs {
		bz, err := cdc.Amino.MarshalBinaryBare(msg)
		require.NoError(t, err)

		decodedMsg, ok := msg.(proto.Message)
		require.True(t, ok)

		err = cdc.Amino.UnmarshalBinaryBare(bz, decodedMsg)
		require.NoError(t, err)

		require.Equal(t, msg, decodedMsg)
	}
}
