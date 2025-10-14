package gosdktest

import (
	"google.golang.org/grpc"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/gosdk"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"

	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

type Blockchain struct {
	GrpcConn *grpc.ClientConn
	Cfg      *testnetwork.Config
	Network  *testnetwork.Network
	Val      *testnetwork.Validator
}

func CreateBlockchain(s *suite.Suite) (nibiru Blockchain, err error) {
	gosdk.EnsureNibiruPrefix()
	genState := genesis.NewTestGenesisState(app.MakeEncodingConfig().Codec)
	cliCfg := testnetwork.BuildNetworkConfig(genState)
	cfg := &cliCfg
	cfg.NumValidators = 1

	network := testnetwork.New(s, *cfg)
	network.WaitForNextBlock()

	val := network.Validators[0]
	AbsorbServerConfig(cfg, &val.AppConfig.Config)
	AbsorbTmConfig(cfg, val.Ctx.Config)

	return Blockchain{
		GrpcConn: val.GrpcClientConn(),
		Cfg:      cfg,
		Network:  network,
		Val:      val,
	}, err
}

func AbsorbServerConfig(
	cfg *testnetwork.Config, srvCfg *serverconfig.Config,
) *testnetwork.Config {
	cfg.GRPCAddress = srvCfg.GRPC.Address
	cfg.APIAddress = srvCfg.API.Address
	return cfg
}

func AbsorbTmConfig(
	cfg *testnetwork.Config, tmCfg *cmtcfg.Config,
) *testnetwork.Config {
	cfg.RPCAddress = tmCfg.RPC.ListenAddress
	return cfg
}

func (chain *Blockchain) TmRpcEndpoint() string {
	return chain.Val.RPCAddress
}
