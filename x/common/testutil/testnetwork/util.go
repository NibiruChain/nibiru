package testnetwork

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	tmtypes "github.com/cometbft/cometbft/abci/types"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/NibiruChain/nibiru/v2/app/codec"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/types"
	tmtime "github.com/cometbft/cometbft/types/time"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

func collectGenFiles(cfg Config, vals []*Validator, outputDir string) error {
	genTime := tmtime.Now()

	for i := 0; i < cfg.NumValidators; i++ {
		tmCfg := vals[i].Ctx.Config

		nodeDir := filepath.Join(outputDir, vals[i].Moniker, "simd")
		gentxsDir := filepath.Join(outputDir, "gentxs")

		tmCfg.Moniker = vals[i].Moniker
		tmCfg.SetRoot(nodeDir)

		initCfg := genutiltypes.NewInitConfig(cfg.ChainID, gentxsDir, vals[i].NodeID, vals[i].PubKey)

		genFile := tmCfg.GenesisFile()
		genDoc, err := types.GenesisDocFromFile(genFile)
		if err != nil {
			return err
		}

		appState, err := genutil.GenAppStateFromConfig(cfg.Codec, cfg.TxConfig,
			tmCfg, initCfg, *genDoc, banktypes.GenesisBalancesIterator{}, genutiltypes.DefaultMessageValidator)
		if err != nil {
			return err
		}

		// overwrite each validator's genesis file to have a canonical genesis time
		if err := genutil.ExportGenesisFileWithTime(genFile, cfg.ChainID, nil, appState, genTime); err != nil {
			return err
		}
	}

	return nil
}

func initGenFiles(cfg Config, genAccounts []authtypes.GenesisAccount, genBalances []banktypes.Balance, genFiles []string) error {
	// set the accounts in the genesis state
	var authGenState authtypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[authtypes.ModuleName], &authGenState)

	accounts, err := authtypes.PackAccounts(genAccounts)
	if err != nil {
		return err
	}

	authGenState.Accounts = append(authGenState.Accounts, accounts...)
	cfg.GenesisState[authtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&authGenState)

	// set the balances in the genesis state
	var bankGenState banktypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[banktypes.ModuleName], &bankGenState)

	bankGenState.Balances = append(bankGenState.Balances, genBalances...)
	cfg.GenesisState[banktypes.ModuleName] = cfg.Codec.MustMarshalJSON(&bankGenState)

	appGenStateJSON, err := json.MarshalIndent(cfg.GenesisState, "", "  ")
	if err != nil {
		return err
	}

	genDoc := types.GenesisDoc{
		ChainID:    cfg.ChainID,
		AppState:   appGenStateJSON,
		Validators: nil,
	}

	// generate empty genesis files for each validator and save
	for i := 0; i < cfg.NumValidators; i++ {
		if err := genDoc.SaveAs(genFiles[i]); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(name string, dir string, contents []byte) error {
	writePath := filepath.Join(dir) //nolint:gocritic
	file := filepath.Join(writePath, name)

	err := tmos.EnsureDir(writePath, 0o755)
	if err != nil {
		return err
	}

	err = os.WriteFile(file, contents, 0o644) // nolint: gosec
	if err != nil {
		return err
	}

	return nil
}

// FillWalletFromValidator fills the wallet with some coins that come from the
// validator.
func FillWalletFromValidator(
	addr sdk.AccAddress, balance sdk.Coins, val *Validator, feesDenom string,
) (*sdk.TxResponse, error) {
	rawResp, err := clitestutil.MsgSendExec(
		val.ClientCtx,
		val.Address,
		addr,
		balance,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewInt64Coin(feesDenom, 10000)),
	)
	if err != nil {
		return nil, err
	}
	return txOK(val.ClientCtx.Codec, rawResp.Bytes())
}

func txOK(jsonCodec sdkcodec.JSONCodec, txBytes []byte) (*sdk.TxResponse, error) {
	resp := new(sdk.TxResponse)
	jsonCodec.MustUnmarshalJSON(txBytes, resp)
	if resp.Code != tmtypes.CodeTypeOK {
		return resp, fmt.Errorf("%s", resp.RawLog)
	}

	return resp, nil
}

/*
NewAccount Creates a new account with a random mnemonic, stores the mnemonic in the keyring, and returns the address.

args:
  - network: the network in which to create the account and key
  - uid: a unique identifier to ensure duplicate accounts are not created

ret:
  - addr: the address of the new account
*/
func NewAccount(network *Network, uid string) sdk.AccAddress {
	val := network.Validators[0]

	// create a new user address
	info, _, err := val.ClientCtx.Keyring.NewMnemonic(
		/* uid */ uid,
		/* language */ keyring.English,
		/* hdPath */ sdk.FullFundraiserPath,
		/* big39Passphrase */ "",
		/* algo */ hd.Secp256k1,
	)
	if err != nil {
		panic(err)
	}

	addr, err := info.GetAddress()
	if err != nil {
		panic(err)
	}
	return addr
}

func NewEthAccount(network *Network, uid string) common.Address {
	addr := NewAccount(network, uid)
	return common.BytesToAddress(addr.Bytes())
}

func NewKeyring(t *testing.T) (
	kring keyring.Keyring,
	algo keyring.SignatureAlgo,
	nodeDirName string,
) {
	var cdc sdkcodec.Codec = codec.MakeEncodingConfig().Codec
	kring = keyring.NewInMemory(cdc)
	nodeDirName = t.TempDir()
	algo = hd.Secp256k1
	return kring, algo, nodeDirName
}
