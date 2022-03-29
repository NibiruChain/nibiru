package keeper_test

import (
	"github.com/MatrixDao/matrix/app"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/cosmos/cosmos-sdk/client"

	// "github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	// paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/suite"
)

// TODO: Use with keeper tests.
type KeeperTestSuite struct {
	suite.Suite
	ctx sdk.Context
	app *app.MatrixApp

	clientCtx   client.Context
	queryClient types.QueryClient
}
