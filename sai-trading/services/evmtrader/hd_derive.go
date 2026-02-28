package evmtrader

import (
	"fmt"

	"github.com/NibiruChain/nibiru/v2/eth"
	nibiruhd "github.com/NibiruChain/nibiru/v2/eth/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type DerivedAccounts struct {
	EthAddrHex    common.Address
	EthAddrBech32 string

	EthPrivateKeyHex string
}

// DeriveAccountsFromMnemonic derives Ethereum account from a mnemonic (using BIP44 path m/44'/60'/0'/0/0)
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

	return &DerivedAccounts{
		EthAddrHex:       ethAddrHex,
		EthAddrBech32:    ethAddrBech32,
		EthPrivateKeyHex: ethPrivKeyHex,
	}, nil
}
