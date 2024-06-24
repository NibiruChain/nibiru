package eth

import (
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

/////////// HexAddr

// TODO: UD-DEBUG: The eth.Addr should map to only one HexString
// TODO: UD-DEBUG: The eth.Addr should be derivable from HexString
// TODO: Constructor should be safe (errorable)
// TODO: UD-DEBUG: HexString -> bytes -> eth.Addr == HexString -> eth.Addr
// TODO: UD-DEBUG: validate: HexString === HexString -> HexString(eth.Addr.Hex())

// HexAddr: An ERC55-comlpiant hexadecimal-encoded string representing the 20
// byte address of an Ethereum account.
type HexAddr string

func NewHexAddr(addr EthAddr) HexAddr {
	return HexAddr(addr.Hex())
}

const HexAddrError = "HexAddrError"

func (h HexAddr) Valid() error {
	// Check address encoding bijectivity
	wantAddr := h.ToAddr().Hex() // gethcommon.Address.Hex()
	haveAddr := string(h)        // should be equivalent to ↑

	if !gethcommon.IsHexAddress(haveAddr) || haveAddr != wantAddr {
		return fmt.Errorf(
			"%s: Etherem address is not represented as expected. We have encoding \"%s\" and instead need \"%s\" (gethcommon.Address.Hex)",
			HexAddrError, haveAddr, wantAddr,
		)
	}

	return nil
}

func NewHexAddrFromStr(addr string) HexAddr {
	return HexAddr(gethcommon.HexToAddress(addr).Hex())
}

func (h HexAddr) ToAddr() EthAddr {
	return gethcommon.HexToAddress(string(h))
}

// func (h HexAddr) Bytes() []byte {
// 	return gethcommon.Hex2Bytes(
// 		strings.TrimPrefix(string(h), "0x"),
// 	)
// }

func (h HexAddr) String() string { return h.ToAddr().Hex() }

func (h HexAddr) FromBytes(bz []byte) HexAddr {
	return HexAddr(gethcommon.BytesToAddress(bz).Hex())
}
