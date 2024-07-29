package eth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type CodecTestSuite struct {
	suite.Suite
}

func TestCodecSuite(t *testing.T) {
	suite.Run(t, new(CodecTestSuite))
}

func (suite *CodecTestSuite) TestRegisterInterfaces() {
	type ProtoNameInfo struct {
		ProtoName string
		Interface interface{}
		WantImpls []string
	}
	protoInfos := []ProtoNameInfo{
		{
			ProtoName: "cosmos.auth.v1beta1.AccountI",
			Interface: new(authtypes.AccountI),
			WantImpls: []string{
				"/eth.types.v1.EthAccount",
				"/cosmos.auth.v1beta1.BaseAccount",
				"/cosmos.auth.v1beta1.ModuleAccount",
			},
		},
		{
			ProtoName: "cosmos.auth.v1beta1.GenesisAccount",
			Interface: new(authtypes.GenesisAccount),
			WantImpls: []string{
				"/eth.types.v1.EthAccount",
				"/cosmos.auth.v1beta1.BaseAccount",
				"/cosmos.auth.v1beta1.ModuleAccount",
			},
		},
		{
			ProtoName: "cosmos.tx.v1beta1.TxExtensionOptionI",
			Interface: new(sdktx.TxExtensionOptionI),
			WantImpls: []string{
				TYPE_URL_WEB3_TX,
			},
		},
	}

	// -------------------------------------------
	// Case 1: Setup: Register all interfaces under test
	// -------------------------------------------
	registry := codectypes.NewInterfaceRegistry()
	for _, protoInfo := range protoInfos {
		registry.RegisterInterface(protoInfo.ProtoName, protoInfo.Interface)
	}
	RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	sdktx.RegisterInterfaces(registry)

	// Test: Assert that all expected protobuf interface implementations are
	// registered (base + Ethereum)
	for _, protoInfo := range protoInfos {
		gotImpls := registry.ListImplementations(protoInfo.ProtoName)
		suite.Require().ElementsMatch(protoInfo.WantImpls, gotImpls)
	}

	// -------------------------------------------
	// Case 2: Setup: Register only eth interfaces
	// -------------------------------------------
	registry = codectypes.NewInterfaceRegistry()
	for _, protoInfo := range protoInfos {
		registry.RegisterInterface(protoInfo.ProtoName, protoInfo.Interface)
	}
	RegisterInterfaces(registry)

	// Test: Assert that all expected protobuf interface implementations are
	// registered (Ethereum only)
	for _, protoInfo := range protoInfos {
		gotImpls := registry.ListImplementations(protoInfo.ProtoName)
		wantImpls := filterImplsForEth(protoInfo.WantImpls)
		suite.Require().ElementsMatch(wantImpls, gotImpls)
	}
}

func filterImplsForEth(implTypeUrls []string) []string {
	typeUrls := []string{}
	for _, typeUrl := range implTypeUrls {
		if strings.Contains(typeUrl, "eth") {
			typeUrls = append(typeUrls, typeUrl)
		}
	}
	return typeUrls
}
