package testutil

import (
	"context"
	"testing"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// SetupClientCtx configures the client and server contexts and returns the
// resultant 'context.Context'. This is useful for executing CLI commands.
func SetupClientCtx(t *testing.T) context.Context {
	home := t.TempDir()
	logger := log.NewNopLogger()
	cfg, err := genutiltest.CreateDefaultTendermintConfig(home)
	require.NoError(t, err)

	appCodec := moduletestutil.MakeTestEncodingConfig().Codec
	var testModuleBasicManager = module.NewBasicManager(genutil.AppModuleBasic{})
	err = genutiltest.ExecInitCmd(
		testModuleBasicManager, home, appCodec)
	require.NoError(t, err)

	serverCtx := server.NewContext(viper.New(), cfg, logger)
	clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home)

	ctx := context.Background()
	ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
	ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)
	return ctx
}
