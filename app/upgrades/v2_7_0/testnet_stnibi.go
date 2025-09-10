package v2_7_0

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_5_0"
)

// TESTNET_STNIBI_ADDR is the (real) hex address of stNIBI on testnet.
// Produced from `nibid q evm funtoken | jq`
// ```
//
//	{
//	  "fun_token": {
//	    "erc20_addr": "0xb6Ec473BeE85DC99B1B350510f592b80F034b5DD",
//	    "bank_denom": "tf/nibi1keqw4dllsczlldd7pmzp25wyl04fw5anh3wxljhg4fjuqj9xnxuqa82rpg/ampNIBIT2",
//	    "is_made_from_coin": true
//	  }
//	}
//
// ```
var TESTNET_STNIBI_ADDR = gethcommon.HexToAddress("0xb6Ec473BeE85DC99B1B350510f592b80F034b5DD")

// Queried from Testnet (ETH Chain ID 6911) on 2025-09-10.
func OldTestnetStnibi() bank.Metadata {
	stnibiDenomTestnet := "tf/nibi1keqw4dllsczlldd7pmzp25wyl04fw5anh3wxljhg4fjuqj9xnxuqa82rpg/ampNIBIT2"
	return bank.Metadata{
		Description: "",
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    stnibiDenomTestnet,
				Exponent: 0,
			},
		},
		Base:    stnibiDenomTestnet,
		Display: stnibiDenomTestnet,
		Name:    stnibiDenomTestnet,
		Symbol:  stnibiDenomTestnet,
		URI:     "",
		URIHash: "",
	}
}

func UpgradeStNibiContractOnTestnet(
	keepers *keepers.PublicKeepers,
	ctx sdk.Context,
) error {
	// Only run this for testnet (6911).
	if keepers.EvmKeeper.EthChainID(ctx).
		Cmp(big.NewInt(appconst.ETH_CHAIN_ID_TESTNET_2)) != 0 {
		return nil // Early return success
	}

	// This is safe to run because we've used this function to upgrade mainnet
	// already.
	return v2_5_0.UpgradeStNibiEvmMetadata(keepers, ctx, TESTNET_STNIBI_ADDR)
}
