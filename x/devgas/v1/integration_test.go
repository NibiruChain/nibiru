package devgas_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	junoapp "github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

// returns context and an app with updated mint keeper
func CreateTestApp(t *testing.T, isCheckTx bool) (*junoapp.NibiruApp, sdk.Context) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	return app, ctx
}
