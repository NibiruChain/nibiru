package wasmbinding

import (
	perpv2keeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type ExecutorPerp struct {
	PerpV2 perpv2keeper.Keeper
}

func (exec *ExecutorPerp) MsgServer() perpv2types.MsgServer {
	return perpv2keeper.NewMsgServerImpl(exec.PerpV2)
}
