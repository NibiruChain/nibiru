package eth

import (
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

/////////// HexAddr

// HexAddr: An ERC55-comlpiant hexadecimal-encoded string representing the 20
// byte address of an Ethereum account.
type HexAddr string

func NewHexAddr(addr EthAddr) HexAddr {
	return HexAddr(addr.Hex())
}

func NewHexAddrFromStr(addr string) (HexAddr, error) {
	hexAddr := HexAddr(gethcommon.HexToAddress(addr).Hex())
	if !gethcommon.IsHexAddress(addr) {
		return hexAddr, fmt.Errorf(
			"%s: input \"%s\" is not an ERC55-compliant, 20 byte hex address",
			HexAddrError, addr,
		)
	}
	return hexAddr, hexAddr.Valid()
}

const HexAddrError = "HexAddrError"

func (h HexAddr) Valid() error {
	// Check address encoding bijectivity
	wantAddr := h.ToAddr().Hex() // gethcommon.Address.Hex()
	haveAddr := string(h)        // should be equivalent to ↑

	if !gethcommon.IsHexAddress(haveAddr) || haveAddr != wantAddr {
		return fmt.Errorf(
			"%s: Ethereum address is not represented as expected. We have encoding \"%s\" and instead need \"%s\" (gethcommon.Address.Hex)",
			HexAddrError, haveAddr, wantAddr,
		)
	}

	return nil
}

func (h HexAddr) ToAddr() EthAddr {
	return gethcommon.HexToAddress(string(h))
}

// ToBytes gets the string representation of the underlying address.
func (h HexAddr) ToBytes() []byte {
	return h.ToAddr().Bytes()
}

func (h HexAddr) String() string { return h.ToAddr().Hex() }
