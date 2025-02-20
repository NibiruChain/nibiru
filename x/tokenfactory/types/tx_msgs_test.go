package types_test

import (
	fmt "fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func TestTxMsgInterface(t *testing.T) {
	creator := testutil.AccAddress().String()
	subdenom := testutil.RandLetters(4)
	denomStr := fmt.Sprintf("tf/%s/%s", creator, subdenom)
	for _, msg := range []legacytx.LegacyMsg{
		&types.MsgCreateDenom{
			Sender:   creator,
			Subdenom: subdenom,
		},
		&types.MsgChangeAdmin{
			Sender:   creator,
			Denom:    denomStr,
			NewAdmin: testutil.AccAddress().String(),
		},
	} {
		t.Run(sdk.MsgTypeURL(msg), func(t *testing.T) {
			require.NotPanics(t, func() {
				_ = msg.GetSignBytes()
			})
		})
	}
}
