package gosdk

import (
	"testing"

	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"

	tmconfig "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

type Blockchain struct {
	GrpcConn *grpc.ClientConn
	Cfg      *cli.Config
	Network  *cli.Network
	Val      *cli.Validator
}

func CreateBlockchain(t *testing.T) (nibiru Blockchain, err error) {
	EnsureNibiruPrefix()
	encConfig := app.MakeEncodingConfig()
	genState := genesis.NewTestGenesisState(encConfig)
	cliCfg := cli.BuildNetworkConfig(genState)
	cfg := &cliCfg
	cfg.NumValidators = 1

	network, err := cli.New(
		t,
		t.TempDir(),
		*cfg,
	)
	if err != nil {
		return nibiru, err
	}
	err = network.WaitForNextBlock()
	if err != nil {
		return nibiru, err
	}

	val := network.Validators[0]
	AbsorbServerConfig(cfg, val.AppConfig)
	AbsorbTmConfig(cfg, val.Ctx.Config)

	grpcConn, err := ConnectGrpcToVal(val)
	if err != nil {
		return nibiru, err
	}
	return Blockchain{
		GrpcConn: grpcConn,
		Cfg:      cfg,
		Network:  network,
		Val:      val,
	}, err
}

func ConnectGrpcToVal(val *cli.Validator) (*grpc.ClientConn, error) {
	grpcUrl := val.AppConfig.GRPC.Address
	return GetGRPCConnection(
		grpcUrl, true, 5,
	)
}

func AbsorbServerConfig(
	cfg *cli.Config, srvCfg *serverconfig.Config,
) *cli.Config {
	cfg.GRPCAddress = srvCfg.GRPC.Address
	cfg.APIAddress = srvCfg.API.Address
	return cfg
}

func AbsorbTmConfig(
	cfg *cli.Config, tmCfg *tmconfig.Config,
) *cli.Config {
	cfg.RPCAddress = tmCfg.RPC.ListenAddress
	return cfg
}

func (chain *Blockchain) TmRpcEndpoint() string {
	return chain.Val.RPCAddress
}
