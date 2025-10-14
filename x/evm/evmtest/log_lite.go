package evmtest

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// Constructs an ERC20 Transfer event
func LogLiteEventErc20Transfer(
	contract gethcommon.Address,
	from, to gethcommon.Address, amt *big.Int,
) evm.LogLite {
	signature := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex()
	return evm.LogLite{
		Address: contract.Hex(),
		Topics: []string{
			signature,
			gethcommon.BytesToHash(from.Bytes()).Hex(),
			gethcommon.BytesToHash(to.Bytes()).Hex(),
		},
		Data: gethcommon.LeftPadBytes(amt.Bytes(), 32),
	}
}

// Constructs a WNIBI.sol Deposit event
func LogLiteEventWnibiDeposit(contract, dst gethcommon.Address, amt *big.Int) evm.LogLite {
	signature := crypto.Keccak256Hash([]byte("Deposit(address,uint256)")).Hex()
	return evm.LogLite{
		Address: contract.Hex(),
		Topics: []string{
			signature,
			gethcommon.BytesToHash(dst.Bytes()).Hex(),
		},
		Data: gethcommon.LeftPadBytes(amt.Bytes(), 32),
	}
}
