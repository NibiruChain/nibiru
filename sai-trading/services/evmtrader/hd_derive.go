package evmtrader

import (
	"fmt"

	"github.com/NibiruChain/nibiru/v2/eth"
	nibiruhd "github.com/NibiruChain/nibiru/v2/eth/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type DerivedAccounts struct {
	CosmosAddrBech32 string
	CosmosAddrHex    common.Address

	EthAddrHex    common.Address
	EthAddrBech32 string

	EthPrivateKeyHex    string
	CosmosPrivateKeyHex string
}

// DeriveAccountsFromMnemonic derives both Ethereum and Cosmos accounts from a mnemonic
func DeriveAccountsFromMnemonic(mnemonic string, bech32Prefix string) (*DerivedAccounts, error) {
	if bech32Prefix == "" {
		bech32Prefix = "nibi" // Default to Nibiru prefix
	}

	ethPrivKeyBytes, err := nibiruhd.EthSecp256k1.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, eth.BIP44HDPath)
	if err != nil {
		return nil, fmt.Errorf("derive ETH private key: %w", err)
	}

	ethPrivKey, err := crypto.ToECDSA(ethPrivKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("convert to ECDSA: %w", err)
	}

	ethAddrHex := crypto.PubkeyToAddress(ethPrivKey.PublicKey)
	ethPrivKeyHex := fmt.Sprintf("%x", crypto.FromECDSA(ethPrivKey))

	ethAddrBech32 := eth.EthAddrToNibiruAddr(ethAddrHex).String()

	cosmosHDPath := sdk.GetConfig().GetFullBIP44Path()
	cosmosPrivKeyBytes, err := hd.Secp256k1.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, cosmosHDPath)
	if err != nil {
		return nil, fmt.Errorf("derive Cosmos private key: %w", err)
	}

	cosmosPrivKey := &secp256k1.PrivKey{Key: cosmosPrivKeyBytes}
	cosmosPrivKeyHex := fmt.Sprintf("%x", cosmosPrivKeyBytes)

	cosmosPubKey := cosmosPrivKey.PubKey()
	cosmosAddrBech32 := sdk.AccAddress(cosmosPubKey.Address()).String()
	cosmosAddrHex := common.BytesToAddress(cosmosPubKey.Address().Bytes())

	return &DerivedAccounts{
		CosmosAddrBech32:    cosmosAddrBech32,
		CosmosAddrHex:       cosmosAddrHex,
		EthAddrHex:          ethAddrHex,
		EthAddrBech32:       ethAddrBech32,
		EthPrivateKeyHex:    ethPrivKeyHex,
		CosmosPrivateKeyHex: cosmosPrivKeyHex,
	}, nil
}
