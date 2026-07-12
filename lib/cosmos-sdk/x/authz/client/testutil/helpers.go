package authz

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil"
	clitestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/cli"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz/client/cli"
)

func CreateGrant(clientCtx client.Context, args []string) (testutil.BufferWriter, error) {
	cmd := cli.NewCmdGrantAuthorization()
	return clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
}
